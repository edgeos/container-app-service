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
  }

  Docker := NewDocker(cfg)
  err = Docker.Init()
  if err != nil {
      t.Error("Expected nil")

  }
}
func TestNewDocker(t *testing.T) {
  cfg, err := config.NewConfig(filePath)
  if err != nil {
      t.Error("Config File Creation Error")
  }
  Docker := NewDocker(cfg)
  if Docker == nil {
    t.Error("Docker is not created")
  }
}

// func TestInit(t *testing.T) {
//   // cfg, err := config.NewConfig(filePath)
//   // if err != nil {
//   //   t.Error("Config File Creation Error")
//   // }
//   var cfg config.Config
//   Docker := NewDocker(cfg)
//   err  := Docker.Start(cfg)
//   if err != nil {
//     t.Error("Start Error")
//   }
// }
