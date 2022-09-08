package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/jetstack/jsctl/internal/command"
)

// Values injected at build-time
var (
	version string
)

func main() {
	cmd := command.Command()
	cmd.Version = version

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
