package provider

import (
	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
	"testing"
)

var filePath = "../ecs.json"

func TestInit(t *testing.T) {
	cfg, err := config.NewConfig(filePath)
	if err != nil {
		t.Error("Config File Creation Error")
		t.Fail()
	}

	Docker := NewDocker(cfg)
	err = Docker.Init()
	if err != nil {
		t.Error("Expected nil")
		t.Fail()
	}
}

func TestNewDocker(t *testing.T) {
	cfg, err := config.NewConfig(filePath)
	if err != nil {
		t.Error("Config File Creation Error")
		t.Fail()
	}
	Docker := NewDocker(cfg)
	if Docker == nil {
		t.Error("Docker is not created")
		t.Fail()
	}
}
