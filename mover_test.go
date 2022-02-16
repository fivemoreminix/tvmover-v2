package main

import (
	"regexp"
	"testing"
)

var testConfig = NewConfig()
var testRegexp = regexp.MustCompile(testConfig.FilenameRegex)

func TestCleanTitle(t *testing.T) {
	if s := CleanTitle("doctor.who", ".-_"); s != "Doctor Who" {
		t.Error(s)
	}
	if s := CleanTitle("this-is._my_.title", ".-_"); s != "This Is My Title" {
		t.Error(s)
	}
}

func TestNewFileMeta(t *testing.T) {
	meta := NewFileMeta("doctor.who.season.1.episode.3", testRegexp, testConfig.IgnoreChars)
	if meta.CleanedTitle != "Doctor Who" {
		t.Error(meta.CleanedTitle)
	}
	if meta.Season != 1 {
		t.Error(meta.Season)
	}
	if meta.Episode != 3 {
		t.Error(meta.Episode)
	}
}
