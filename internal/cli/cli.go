package cli

import (
	"fmt"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/urfave/cli/v2"
)

var app = cli.NewApp()
var foxbox client.Client

func init() {
	app.Usage = "A simple, cli-based container runtime"
}

func Start(args []string) (err error) {
	foxbox, err = client.FromUser()
	if err != nil {
		return fmt.Errorf("initializing foxbox client: %w", err)
	}

	return app.Run(args)
}
