package provider

import (
	"io"

	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
	"github.build.ge.com/PredixEdgeOS/container-app-service/types"
)

// Provider : Functions that a provider must include
type Provider interface {
	Init() error

	Deploy(metadata types.Metadata, file io.Reader, persistent bool) (*types.App, error)
	Undeploy(id string) error
	PurgePersistent(name string) error
	Kill(id string) error

	Start(id string) error
	Stop(id string) error
	Restart(id string) error

	GetApplication(id string) (*types.AppDetails, error)
	ListApplications() types.Applications
	ListPersistentApplications() types.PersistentApps
}

// NewProvider ...
func NewProvider(c config.Config) Provider {
	p := NewDocker(c)
	if err := p.Init(); err != nil {
		return nil
	}
	return p
}
