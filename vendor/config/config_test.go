package config

import (
  "testing"
)

func TestNewConfig(t *testing.T) {
  var filePath string = "../ecs.json"
  _, err := NewConfig(filePath)
  if err != nil {
    t.Error("Failed to create new config!")
    t.Fail()
  }
}
