package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	rootDir, err := os.MkdirTemp(os.TempDir(), "tmp-test-docs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(rootDir); err != nil {
			t.Fatal(err)
		}
	}()

	tests := map[string]struct {
		input   []string
		expDirs []string
		expErr  bool
	}{
		"if no arguments given should error": {
			input:  []string{"cobra"},
			expErr: true,
		},
		"if two arguments given should error": {
			input:  []string{"cobra", "foo", "bar"},
			expErr: true,
		},
		"if directory given, should write docs": {
			input:   []string{"cobra", filepath.Join(rootDir, "foo")},
			expDirs: []string{"foo/jsctl.md", "foo/jsctl_operator.md", "foo/jsctl_operator_installations.md"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := run(test.input)
			if test.expErr != (err != nil) {
				t.Errorf("got unexpected error, exp=%t got=%v",
					test.expErr, err)
			}

			for _, dir := range test.expDirs {
				if _, err := os.Stat(filepath.Join(rootDir, dir)); err != nil {
					t.Errorf("stat error on expected directory: %s", err)
				}
			}
		})
	}
}
