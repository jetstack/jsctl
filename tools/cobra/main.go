package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jetstack/jsctl/internal/command"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
)

// This code is largely adapted from https://github.com/cert-manager/cert-manager/tree/master/tools/cobra

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(args []string) error {
	if len(args) != 2 {
		return errors.New("expecting single output directory argument")
	}

	// remove all global flags that are imported in
	pflag.CommandLine = nil

	dir, err := homedir.Expand(args[1])
	if err != nil {
		return err
	}

	if err := ensureDirectory(dir); err != nil {
		return err
	}

	c := command.Command()
	f, err := os.Create(fmt.Sprintf("%s/README.md", dir))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := doc.GenMarkdown(c, f); err != nil {
		return err
	}

	if err := doc.GenMarkdownTree(c, dir); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "jsctl documentation generated at %s\n", dir)

	return nil
}

func ensureDirectory(dir string) error {
	s, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Mkdir(dir, os.FileMode(0755))
		}
		return err
	}

	if !s.IsDir() {
		return fmt.Errorf("path it not directory: %s", dir)
	}

	return nil
}
