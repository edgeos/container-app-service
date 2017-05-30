package provider

import (
	"io"

	"github.build.ge.com/container-app-service/config"
	"github.build.ge.com/container-app-service/types"
)

// Functions that a provider must include
type Provider interface {
	Init() error

	Deploy(metadata types.Metadata, file io.Reader) (*types.App, error)
	Undeploy(id string) error

	Start(id string) error
	Stop(id string) error
	Restart(id string) error

	GetApplication(id string) (*types.AppDetails, error)
	ListApplications() types.Applications
}

func NewProvider(c config.Config) Provider {
	return NewDocker(c)
}
