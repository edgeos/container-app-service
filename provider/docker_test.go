package provider

import (
  "testing"
  "github.build.ge.com/PredixEdgeOS/container-app-service/config"
)

var filePath string = "../ecs.json"

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