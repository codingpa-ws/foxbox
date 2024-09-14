package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func init() {
	imageCommand.Subcommands = append(imageCommand.Subcommands, &cli.Command{
		Name:      "ls",
		Usage:     "List all images",
		Action:    imageLs,
		UsageText: "[name...]",
	})
}

func imageLs(ctx *cli.Context) (err error) {
	image, err := foxbox.ListImages()
	if err != nil {
		return
	}

	if len(image) == 0 {
		return fmt.Errorf("no images found in %s", storage.ImageBase())
	}

	for _, v := range image {
		fmt.Println(v.Name)
	}

	return
}
