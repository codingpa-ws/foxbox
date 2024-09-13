package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func init() {
	app.Commands = append(app.Commands, &cli.Command{
		Name:      "rm",
		Usage:     "Remove a foxbox",
		Action:    rm,
		UsageText: "[name...]",
	})
}

func rm(ctx *cli.Context) (err error) {
	for _, id := range ctx.Args().Slice() {
		err := foxbox.Delete(id, nil)
		if err != nil {
			return fmt.Errorf("removing %s: %w", id, err)
		}
	}

	return nil
}
