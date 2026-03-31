package main

import (
	"testing"
)

func TestMigrate_Up_ParsesArgs(t *testing.T) {
	args := []string{"migrate", "-up", "-dir", "migrations"}
	dir := getMigrationsDir(args)
	if dir != "migrations" {
		t.Errorf("expected migrations dir 'migrations', got %q", dir)
	}
}

func TestMigrate_Down_ParsesArgs(t *testing.T) {
	args := []string{"migrate", "-down", "-dir", "migrations"}
	dir := getMigrationsDir(args)
	if dir != "migrations" {
		t.Errorf("expected migrations dir 'migrations', got %q", dir)
	}
}

func TestMigrate_GetCommand(t *testing.T) {
	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{"migrate", "-up"}, "up"},
		{[]string{"migrate", "-down"}, "down"},
		{[]string{"migrate"}, ""},
	}

	for _, tt := range tests {
		cmd := getCommand(tt.args)
		if cmd != tt.expected {
			t.Errorf("args %v: expected command %q, got %q", tt.args, tt.expected, cmd)
		}
	}
}
