package client

import (
	"context"
	"os"
	"testing"
)

// I've not done any sensible testing, since we're just testing out if it'll work like we planned...this is more like a run environment....

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestClient(t *testing.T) {
	_, err := NewClient(context.Background())
	if err != nil {
		t.Error(err)
	}
}