package main

import (
	"os"
	"regexp"
	"testing"
)

func TestVersionIsTheReleaseVersion(t *testing.T) {
	if version == "dev" {
		t.Skip("version is set by -ldflags at build time")
	}
	raw, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?m)^## \[([0-9][^\]]*)\]`).FindSubmatch(raw)
	if match == nil {
		t.Fatal("CHANGELOG.md has no released version heading")
	}
	if want := "v" + string(match[1]); version != want {
		t.Errorf("version = %q, want %q", version, want)
	}
}
