// +build unit

package main

import (
	"testing"
	"time"
)

func TestHelpCheckErr(t *testing.T) {
	cmd := "nohelp"

	if helpCheck(cmd) {
		t.Errorf("There is no help in cmd %s", cmd)
	}
}

func TestHelpCheckOk(t *testing.T) {
	cmd := "--help"

	if !helpCheck(cmd) {
		t.Errorf("There is no help in cmd %s", cmd)
	}

	cmd = "help"

	if !helpCheck(cmd) {
		t.Errorf("There is no help in cmd %s", cmd)
	}

	cmd = "-h"

	if !helpCheck(cmd) {
		t.Errorf("There is no help in cmd %s", cmd)
	}
}

func TestFileCheckErr(t *testing.T) {
	cmd := "---file"

	if fileCheck(cmd) {
		t.Errorf("There is no file in cmd %s", cmd)
	}
}

func TestFileCheckOk(t *testing.T) {
	cmd := "--file"

	if !fileCheck(cmd) {
		t.Errorf("There is no file in cmd %s", cmd)
	}

	cmd = "-f"

	if !fileCheck(cmd) {
		t.Errorf("There is no file in cmd %s", cmd)
	}

}

func TestParseCmdErr(t *testing.T) {
	target := "some"
	args := []string{}

	cfg := &SchedCtlConfig{
		ConnectionTimeout: 5 * time.Second,
	}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}

	target = "some"
	args = []string{"help"}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}

	target = "schedctl"
	args = []string{"workers"}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}

	target = "schedctl"
	args = []string{"tasks"}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}

	target = "tasks"
	args = []string{"tasks"}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}

	target = "workers"
	args = []string{"schedule"}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}

	target = "schedctl"
	args = []string{"ls"}

	if err := parseCmd(target, args, cfg); err == nil {
		t.Errorf("Ther should be an error with this args %+v and target %s ",
			args, target)
	}
}
