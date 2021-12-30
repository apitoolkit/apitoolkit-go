package server

import (
	"os"
	"testing"
)

// I've not done any sensible testing, since we're just testing out if it'll work like we planned...this is more like a run environment....

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	RunServer()
}