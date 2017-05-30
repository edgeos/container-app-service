package provider

import (
	"errors"
	"io"
	"os"

	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/docker/ctx"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/options"
	"golang.org/x/net/context"

	"github.build.ge.com/container-app-service/config"
	"github.build.ge.com/container-app-service/types"
	"github.build.ge.com/container-app-service/utils"
)

type Docker struct {
	cfg  config.Config
	apps map[string]types.App
}

func NewDocker(c config.Config) *Docker {
	provider := new(Docker)
	provider.cfg = c
	provider.apps = make(map[string]types.App)
	return provider
}

func (p *Docker) Init() error {
	return nil
}

func (p *Docker) Deploy(metadata types.Metadata, file io.Reader) (*types.App, error) {
	var err error
	var uuid string
	if uuid, err = utils.NewUUID(); err == nil {
		path := p.cfg.DataVolume + "/" + uuid
		os.Mkdir(path, os.ModePerm)
		utils.Unpack(file, path)
		composeFile := path + "/docker-compose.yml"

		c := ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  uuid,
			},
		}

		var prj project.APIProject
		if prj, err = docker.NewProject(&c, nil); err == nil {
			p.apps[uuid] = types.App{
				UUID:    uuid,
				Name:    metadata.Name,
				Version: metadata.Version,
				Path:    path}

			utils.Save(p.cfg.DataVolume+"/application.json", p.apps)

			err = prj.Up(context.Background(), options.Up{})
			if err == nil {
				app := p.apps[uuid]
				return &app, nil
			}

			return nil, err
		}
	}

	return nil, errors.New(types.InvalidID)
}

func (p *Docker) Undeploy(id string) error {
	app, exists := p.apps[id]
	if exists {
		composeFile := app.Path + "/docker-compose.yml"
		prj, _ := docker.NewProject(&ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  app.UUID,
			},
		}, nil)

		prj.Down(context.Background(), options.Down{})
		os.RemoveAll(app.Path)
		delete(p.apps, app.UUID)
		utils.Save("/tmp/application.json", p.apps)
	}

	return errors.New(types.InvalidID)
}

func (p *Docker) Start(id string) error {
	var err error
	app, exists := p.apps[id]
	if exists {
		composeFile := app.Path + "/docker-compose.yml"
		prj, _ := docker.NewProject(&ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  app.UUID,
			},
		}, nil)

		if err = prj.Up(context.Background(), options.Up{}); err == nil {
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

func (p *Docker) Stop(id string) error {
	var err error
	app, exists := p.apps[id]
	if exists {
		composeFile := app.Path + "/docker-compose.yml"
		prj, _ := docker.NewProject(&ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  app.UUID,
			},
		}, nil)

		if err = prj.Down(context.Background(), options.Down{}); err == nil {
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

func (p *Docker) Restart(id string) error {
	var err error
	app, exists := p.apps[id]
	if exists {
		composeFile := app.Path + "/docker-compose.yml"
		prj, _ := docker.NewProject(&ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  app.UUID,
			},
		}, nil)

		prj.Down(context.Background(), options.Down{})
		if err = prj.Up(context.Background(), options.Up{}); err == nil {
			return nil
		}
		return err
	}

	return errors.New(types.InvalidID)
}

func (p *Docker) GetApplication(id string) (*types.AppDetails, error) {
	app, exists := p.apps[id]
	if exists {
		composeFile := app.Path + "/docker-compose.yml"
		prj, _ := docker.NewProject(&ctx.Context{
			Context: project.Context{
				ComposeFiles: []string{composeFile},
				ProjectName:  app.UUID,
			},
		}, nil)

		var err error
		var info project.InfoSet
		if info, err = prj.Ps(context.Background()); err == nil {
			var details types.AppDetails
			details.UUID = app.UUID
			details.Name = app.Name
			details.Version = app.Version
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
	var response types.Applications
	for k := range p.apps {
		response.Apps = append(response.Apps, p.apps[k])
	}

	return response
}
