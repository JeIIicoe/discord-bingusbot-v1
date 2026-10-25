package main

import "testing"

func TestRunRequiresToken(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "")
	if err := run(); err == nil {
		t.Error("run() error = nil, want error when DISCORD_TOKEN is not set")
	}
}
