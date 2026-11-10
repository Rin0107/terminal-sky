package app

import (
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var b strings.Builder
	if err := Run([]string{"-version"}, &b); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(b.String()) != Version {
		t.Fatalf("got %q", b.String())
	}
}

func TestRunRejectsUnknownArgs(t *testing.T) {
	if err := Run([]string{"nope"}, &strings.Builder{}); err == nil {
		t.Fatal("知らない引数を通した")
	}
}
