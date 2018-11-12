package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	filePath := "../ecs.json"
	_, err := NewConfig(filePath)
	if err != nil {
		t.Error("Failed to create new config!")
		t.Fail()
	}
}
