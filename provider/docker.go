package provider

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"io/ioutil"
	"strings"
	"encoding/json"

	"github.com/docker/docker/client"

	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/docker/ctx"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/events"
	"github.com/docker/libcompose/project/options"
	"golang.org/x/net/context"

	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
	"github.build.ge.com/PredixEdgeOS/container-app-service/types"
	"github.build.ge.com/PredixEdgeOS/container-app-service/utils"
)

// ComposeApp ...
type ComposeApp struct {
	Info    types.App                  `json:"info"`
	Client  project.APIProject         `json:"-"`
	Events  chan events.ContainerEvent `json:"-"`
	Monitor bool                       `json:"-"`
	Active  bool                       `json:"-"`
}

// Docker ...
type Docker struct {
	Cfg  config.Config
	Apps map[string]*ComposeApp
	Lock sync.RWMutex
	IsHealthyMap map[string](map[string]bool)
}

// EventListener ...
type EventListener struct {
	provider *Docker
}

// NewListener ...
func NewListener(d *Docker) {
	l := EventListener{
		provider: d,
	}
	go l.start()
}

// LoadImage loads a docker image from a tar ball file
func LoadImage(infilePath *string) error {
	input, err := os.Open(*infilePath)
	if err == nil {
		defer input.Close()
		cli, err := client.NewEnvClient()
		if err == nil {
			imageLoadResponse, err := cli.ImageLoad(context.Background(), input, false)
			if imageLoadResponse.JSON == false {
				return errors.New("expected a JSON response from ImageLoad() function , was not.")
			}
			body, err := ioutil.ReadAll(imageLoadResponse.Body)
			if err != nil {
				return err
			}
			// Docker returns a new line separated list of json, so iterate over it and check for errors
			for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
				var lineJson interface{}
				err = json.Unmarshal([]byte(line), &lineJson)
				if err != nil {
					return err
				}
				if val, ok := lineJson.(map[string]interface{})["error"]; ok {
					return errors.New(string(val.(string)))
				}
			}
			if !strings.Contains(string(body), "Loaded image") {
				time.Sleep(3 * time.Second)
			}
			defer imageLoadResponse.Body.Close()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return err
}

// start ...
func (l *EventListener) start() {
	for id := range l.provider.Apps {
		l.provider.IsHealthyMap[id] = make(map[string]bool)
		app := l.provider.Apps[id].Client.(*project.Project)
		for name, _ := range app.ServiceConfigs.All() {
			l.provider.IsHealthyMap[id][name] = true;
		}
	}

	for {
		l.provider.Lock.RLock()
		for id := range l.provider.Apps {
			eventstream := l.provider.Apps[id].Events
			select {
				case event := <-eventstream:
					if l.provider.Apps[id].Active == true && l.provider.Apps[id].Monitor == true {
						if event.Event == "health_status: unhealthy" {
							l.provider.IsHealthyMap[id][event.Service] = false
							l.provider.Apps[id].Client.Restart(context.Background(), 5, event.Service)
						} else if event.Event == "health_status: healthy" {
							l.provider.IsHealthyMap[id][event.Service] = true
						} else if l.provider.IsHealthyMap[id][event.Service] == true && event.Event == "stop" {
							l.provider.Apps[id].Client.Start(context.Background(), event.Service)
						}
					}
				default:
			}
		}
		l.provider.Lock.RUnlock()
		time.Sleep(1000 * time.Millisecond)
	}
}

// NewDocker ...
func NewDocker(c config.Config) *Docker {
	provider := new(Docker)
	provider.Apps = make(map[string]*ComposeApp)
	provider.IsHealthyMap = make(map[string](map[string]bool))
	provider.Cfg = c
	return provider
}

// Init ...app.
func (p *Docker) Init() error {
  var data map[string]ComposeApp
  utils.Load(p.Cfg.DataVolume+"/application.json", &data)
  p.Apps = make(map[string]*ComposeApp)
  for id := range data {
    p.Apps[id] = &ComposeApp{
      Info: types.App{
        UUID:    id,
        Name:    data[id].Info.Name,
        Version: data[id].Info.Version,
        Path:    data[id].Info.Path,
        Monitor: data[id].Info.Monitor,
				Active:  data[id].Info.Active,
      },
      Monitor: strings.EqualFold(data[id].Info.Monitor, "yes"),
      Active:  strings.EqualFold(data[id].Info.Active, "yes"),
    }

    composeFile := p.Apps[id].Info.Path + "/docker-compose.yml"
    c := ctx.Context{
      Context: project.Context{
        ComposeFiles: []string{composeFile},
        ProjectName:  id,
      },
    }

    var err error
    var prj project.APIProject
    if prj, err = docker.NewProject(&c, nil); err == nil {
      p.Apps[id].Client = prj
			if p.Apps[id].Active == true {
				// Stop running docker containers for the app first
				if err = p.Apps[id].Client.Down(context.Background(), options.Down{}); err != nil {
					return err
				}

      	err = prj.Up(context.Background(), options.Up{})
	      if err == nil {
	        eventstream, _ := p.Apps[id].Client.Events(context.Background())
	        p.Apps[id].Events = eventstream
	      }
			}
    } else {
      delete(p.Apps, id)
      utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
    }
  }
	NewListener(p)
  return nil
}

// Deploy ...
func (p *Docker) Deploy(metadata types.Metadata, file io.Reader) (*types.App, error) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	var uuid string
	if uuid, err = utils.NewUUID(); err == nil {
		path := p.Cfg.DataVolume + "/" + uuid
		os.Mkdir(path, os.ModePerm)
		utils.Unpack(file, path)
		composeFile := path + "/docker-compose.yml"

		files, err := ioutil.ReadDir(path)
		if err == nil {
			for _, f := range files {
				if strings.Contains(f.Name(), ".tar") {
					var infile = new(string)
					*infile = path + "/" + f.Name()
					err = LoadImage(infile)
					if err != nil {
					    os.RemoveAll(path)
					    return nil, err
					}
				}
			}
		}

		c := ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  uuid,
			},
		}

		var prj project.APIProject
		prj, err = docker.NewProject(&c, nil)
		if err == nil {
			isMonitor := false
			if strings.EqualFold(metadata.Monitor, "yes") {
				isMonitor = true
				p.IsHealthyMap[uuid] = make(map[string]bool)
				app := prj.(*project.Project)
				for name, _ := range app.ServiceConfigs.All() {
					p.IsHealthyMap[uuid][name] = true;
				}
			}
			p.Apps[uuid] = &ComposeApp{
				Info: types.App{
					UUID:    uuid,
					Name:    metadata.Name,
					Version: metadata.Version,
					Path:    path,
					Monitor: metadata.Monitor,
					Active:	 "no",
				},
				Client:  prj,
				Monitor: isMonitor,
				Active: false,
			}

			err = prj.Up(context.Background(), options.Up{})
			if err == nil {
				eventstream, _ := p.Apps[uuid].Client.Events(context.Background())
				p.Apps[uuid].Events = eventstream
				p.Apps[uuid].Active = true
				p.Apps[uuid].Info.Active = "yes"
				utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
				info := p.Apps[uuid].Info
				return &info, nil
			} else {
				app, _ := p.Apps[uuid]
				app.Client.Down(context.Background(), options.Down{})
				app.Client.Delete(context.Background(), options.Delete{})
				os.RemoveAll(app.Info.Path)
				delete(p.Apps, app.Info.UUID)
				utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
				return nil, err
			}
		} else {
			os.RemoveAll(path)
		}
	}

	return nil, errors.New(types.InvalidID)
}

// Undeploy ...
func (p *Docker) Undeploy(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	app, exists := p.Apps[id]
	if exists {
		app.Client.Down(context.Background(), options.Down{})
		app.Client.Delete(context.Background(), options.Delete{})
		os.RemoveAll(app.Info.Path)
		delete(p.Apps, app.Info.UUID)
		utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)

		return nil
	}

	return errors.New(types.InvalidID)
}

// Start ...
func (p *Docker) Start(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	app, exists := p.Apps[id]
	if exists {
		if err = app.Client.Up(context.Background(), options.Up{}); err == nil {
			p.Apps[id].Active = true
			p.Apps[id].Info.Active = "yes"
			utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

// Stop ...
func (p *Docker) Stop(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	app, exists := p.Apps[id]
	if exists {
		p.Apps[id].Active = false
		p.Apps[id].Info.Active = "no"
		if err = app.Client.Down(context.Background(), options.Down{}); err == nil {
			utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

// Restart ...
func (p *Docker) Restart(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	app, exists := p.Apps[id]
	if exists {
		p.Apps[id].Active = false
		app.Client.Down(context.Background(), options.Down{})
		if err = app.Client.Up(context.Background(), options.Up{}); err == nil {
			p.Apps[id].Active = true
			p.Apps[id].Info.Active = "yes"
			utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

// GetApplication ...
func (p *Docker) GetApplication(id string) (*types.AppDetails, error) {
	p.Lock.RLock()
	defer p.Lock.RUnlock()

	app, exists := p.Apps[id]
	if exists {
		var err error
		var info project.InfoSet
		if info, err = app.Client.Ps(context.Background()); err == nil {
			var details types.AppDetails
			details.UUID = app.Info.UUID
			details.Name = app.Info.Name
			details.Version = app.Info.Version
			details.Monitor = app.Info.Monitor
			for s := range info {
				service := info[s]
				details.Containers = append(details.Containers, types.Container{
					ID:      service["Id"],
					Name:    service["Name"],
					Command: service["Command"],
					State:   service["State"],
					Ports:   service["Ports"]})
			}
			return &details, nil
		}
		return nil, err
	}
	return nil, errors.New(types.InvalidID)
}

// ListApplications ...
func (p *Docker) ListApplications() types.Applications {
	p.Lock.RLock()
	defer p.Lock.RUnlock()

	var response types.Applications
	for k := range p.Apps {
		response.Apps = append(response.Apps, p.Apps[k].Info)
	}

	return response
}
