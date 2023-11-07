package cli

import (
	"github.com/urfave/cli/v2"
)

var app = cli.NewApp()

func init() {
	app.Usage = "A simple, cli-based container runtime"
}

func Start(args []string) error {
	return app.Run(args)
}
