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
	version string = "dev"
	commit  string = "unknown"
	date    string = "unknown"
)

func main() {
	cmd := command.Command()
	cmd.Version = fmt.Sprintf("%s, commit %s, built at %s", version, commit, date)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
