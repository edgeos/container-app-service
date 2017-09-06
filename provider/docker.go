package provider

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"fmt"
	"io/ioutil"
	"context"
	"github.com/docker/docker/client"

	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/docker/ctx"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/events"
	"github.com/docker/libcompose/project/options"
	// "golang.org/x/net/context"

	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
	"github.build.ge.com/PredixEdgeOS/container-app-service/types"
	"github.build.ge.com/PredixEdgeOS/container-app-service/utils"
)

type ComposeApp struct {
	Info    types.App                  `json:"info"`
	Client  project.APIProject         `json:"-"`
	Events  chan events.ContainerEvent `json:"-"`
	Monitor bool                       `json:"-"`
}

type Docker struct {
	Cfg  config.Config
	Apps map[string]*ComposeApp
	Lock sync.RWMutex
}

type EventListener struct {
	provider *Docker
}

func NewListener(d *Docker) {
	l := EventListener{
		provider: d,
	}
	go l.start()
}

func (l *EventListener) start() {
	for {
		l.provider.Lock.RLock()
		for id := range l.provider.Apps {
			eventstream := l.provider.Apps[id].Events

			select {
			case event := <-eventstream:
				if l.provider.Apps[id].Monitor == true {
					if event.Event == "stop" {
						l.provider.Apps[id].Client.Start(context.Background(), event.Service)
					}
				}
			default:
				// do nothing
			}
		}
		l.provider.Lock.RUnlock()
		time.Sleep(100 * time.Millisecond)
	}
}

func NewDocker(c config.Config) *Docker {
	provider := new(Docker)
	provider.Apps = make(map[string]*ComposeApp)
	provider.Cfg = c
	return provider
}

func (p *Docker) Init() error {
	NewListener(p)

	utils.Load(p.Cfg.DataVolume+"/application.json", p.Apps)

	for id := range p.Apps {
		info := p.Apps[id].Info

		composeFile := info.Path + "/docker-compose.yml"
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
			p.Apps[id].Monitor = false
			err = prj.Up(context.Background(), options.Up{})
			if err == nil {
				eventstream, _ := p.Apps[id].Client.Events(context.Background())
				p.Apps[id].Events = eventstream
				p.Apps[id].Monitor = true
			}
		} else {
			delete(p.Apps, id)
			utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)
		}
	}
	return nil
}

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

		// Check if a tar ball file (*.tar.gz file) exists under the unpacked(unzipped) directory/path,
		// if yes then call docker load to turn that file into a docker image
		var infile = new(string)
		*infile = path + "/helloyutao.tar.gz"

		input, err := os.Open(*infile)
		if err != nil {
			panic(err)
		}
		defer input.Close()

		cli, err := client.NewEnvClient()
		if err != nil {
			panic(err)
		}

		imageLoadResponse, err := cli.ImageLoad(context.Background(), input, false)
		if err != nil {
			panic(err)
		}
		defer imageLoadResponse.Body.Close()
		if imageLoadResponse.JSON != true {
			panic("expected a JSON response, was not.")
		} else {
			fmt.Printf("imageLoadResponse.JSON is true." + "\n")
		}
		body, err := ioutil.ReadAll(imageLoadResponse.Body)
		if err != nil {
			panic(err)
		}
		fmt.Printf("imageLoadResponse.Body is " + string(body) + "\n")



		c := ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  uuid,
			},
		}

		var prj project.APIProject
		if prj, err = docker.NewProject(&c, nil); err == nil {
			p.Apps[uuid] = &ComposeApp{
				Info: types.App{
					UUID:    uuid,
					Name:    metadata.Name,
					Version: metadata.Version,
					Path:    path,
				},
				Client:  prj,
				Monitor: false,
			}

			utils.Save(p.Cfg.DataVolume+"/application.json", p.Apps)

			err = prj.Up(context.Background(), options.Up{})
			if err == nil {
				eventstream, _ := p.Apps[uuid].Client.Events(context.Background())
				p.Apps[uuid].Events = eventstream
				p.Apps[uuid].Monitor = true
				info := p.Apps[uuid].Info
				return &info, nil
			}

			return nil, err
		}
	}

	return nil, errors.New(types.InvalidID)
}

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

func (p *Docker) Start(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	app, exists := p.Apps[id]
	if exists {
		if err = app.Client.Up(context.Background(), options.Up{}); err == nil {
			p.Apps[id].Monitor = true
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

func (p *Docker) Stop(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	app, exists := p.Apps[id]
	if exists {
		p.Apps[id].Monitor = false
		if err = app.Client.Down(context.Background(), options.Down{}); err == nil {
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

func (p *Docker) Restart(id string) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var err error
	app, exists := p.Apps[id]
	if exists {
		p.Apps[id].Monitor = false
		app.Client.Down(context.Background(), options.Down{})
		if err = app.Client.Up(context.Background(), options.Up{}); err == nil {
			p.Apps[id].Monitor = true
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

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

func (p *Docker) ListApplications() types.Applications {
	p.Lock.RLock()
	defer p.Lock.RUnlock()

	var response types.Applications
	for k := range p.Apps {
		response.Apps = append(response.Apps, p.Apps[k].Info)
	}

	return response
}
