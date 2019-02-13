// +build unit

package common

import (
	"testing"
)

func TestInitLoggingParseLevelError(t *testing.T) {
	err := InitLogging("some", "/some/path")

	if err == nil {
		t.Error("Error is nil for level some")
	}
}

func TestInitLoggingParseLevel(t *testing.T) {
	err := InitLogging("debug", "/some/path")

	if err != nil {
		t.Errorf("Error is non nil for level debug %s", err.Error())
	}
}
