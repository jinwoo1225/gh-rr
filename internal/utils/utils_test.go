package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHumanizeDuration(t *testing.T) {
	tests := map[int]string{
		0:       "0s",
		30:      "30s",
		59:      "59s",
		60:      "1m",
		3599:    "59m",
		3600:    "1h",
		86399:   "23h",
		86400:   "1d",
		604799:  "6d",
		604800:  "1w",
		1209600: "2w",
	}
	for input, want := range tests {
		if got := HumanizeDuration(input); got != want {
			t.Errorf("HumanizeDuration(%d) = %q; want %q", input, got, want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	origHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	fakeHome := filepath.Join(string(os.PathSeparator), "home", "user")
	os.Setenv("HOME", fakeHome)

	cases := []struct {
		input string
		want  string
	}{
		{"~", fakeHome},
		{"~/dir", filepath.Join(fakeHome, "dir")},
		{"~dir", filepath.Join(fakeHome, "dir")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~/", fakeHome},
	}
	for _, c := range cases {
		if got := ExpandHome(c.input); got != c.want {
			t.Errorf("ExpandHome(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}

func TestGetBaseDir(t *testing.T) {
	origHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	os.Setenv("HOME", filepath.Join(string(os.PathSeparator), "home", "user"))
	origBase := os.Getenv("BASE_DIR")
	t.Cleanup(func() { os.Setenv("BASE_DIR", origBase) })

	os.Unsetenv("BASE_DIR")
	wantDefault := filepath.Join(os.Getenv("HOME"), "workspace")
	if got := GetBaseDir(); got != wantDefault {
		t.Errorf("GetBaseDir() without BASE_DIR = %q; want %q", got, wantDefault)
	}

	os.Setenv("BASE_DIR", "/tmp/base")
	if got := GetBaseDir(); got != "/tmp/base" {
		t.Errorf("GetBaseDir() with absolute BASE_DIR = %q; want %q", got, "/tmp/base")
	}

	os.Setenv("BASE_DIR", "~/base")
	wantHome := filepath.Join(os.Getenv("HOME"), "base")
	if got := GetBaseDir(); got != wantHome {
		t.Errorf("GetBaseDir() with BASE_DIR=~/base = %q; want %q", got, wantHome)
	}
}
