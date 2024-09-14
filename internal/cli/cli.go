package cli

import (
	"fmt"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/codingpa-ws/foxbox/internal/store"
	"github.com/urfave/cli/v2"
)

var app = cli.NewApp()
var foxbox client.Client
var storage *store.Store

func init() {
	app.Usage = "A simple, cli-based container runtime"
}

func Start(args []string) (err error) {
	storage, err = client.GetOrCreateUserStore()
	if err != nil {
		return fmt.Errorf("initializing foxbox user store: %w", err)
	}
	foxbox = client.FromStore(storage)

	return app.Run(args)
}
