package utils

import (
	"testing"
	"time"
)

func TestNewSimpleBackoff(t *testing.T) {
	simpleBackoff := NewSimpleBackoff(time.Second, time.Minute, 0.2, 2)
	if simpleBackoff == nil {
		t.Error("Failed to create a new SimpleBackoff!")
		t.Fail()
	}
}

func TestDuration(t *testing.T) {
	simpleBackoff := NewSimpleBackoff(time.Second, time.Minute, 0.2, 2)
	if simpleBackoff == nil {
		t.Error("Failed to create a new SimpleBackoff!")
		t.Fail()
	}
	duration := simpleBackoff.Duration()
	if duration < 0 {
		t.Error("Failed to in testing Duration() function!")
		t.Fail()
	}
}

func TestAddJitter(t *testing.T) {
	newDuration := AddJitter(time.Second, 0)
	if newDuration.Nanoseconds() != 1000000000 {
		t.Error("Failed to in testing AddJitter() function when Jitter is 0!")
		t.Fail()
	}
	newDuration = AddJitter(time.Second, 99)
	if newDuration.Nanoseconds() < 1000000000 {
		t.Error("Failed to in testing AddJitter() function when Jitter is not zero!")
		t.Fail()
	}
}
