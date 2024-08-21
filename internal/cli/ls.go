package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func init() {
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "ls",
		Usage:  "List all foxboxes",
		Action: ls,
	})
}

func ls(ctx *cli.Context) (err error) {
	ids, err := foxbox.List(nil)
	if err != nil {
		return
	}

	for _, v := range ids {
		fmt.Println(v)
	}

	return
}
