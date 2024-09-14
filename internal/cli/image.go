package cli

import "github.com/urfave/cli/v2"

var imageCommand = &cli.Command{
	Name:  "image",
	Usage: "Image-related commands",
}

func init() {
	app.Commands = append(app.Commands, imageCommand)
}
