package cli

import (
	"fmt"
	"log"

	"github.com/c2h5oh/datasize"
	"github.com/codingpa-ws/foxbox/client"
	"github.com/urfave/cli/v2"
)

func init() {
	app.Commands = append(app.Commands, &cli.Command{
		Name:      "run",
		Usage:     "Run a foxbox with the specified image",
		Action:    run,
		ArgsUsage: "[image] [(command) (args...)]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "rm",
				Usage: "removes the foxbox after execution has finished",
			},
			&cli.BoolFlag{
				Name:  "disable-network",
				Usage: "disables bridge networking (via slirp)",
			},
			&cli.Float64Flag{
				Name:  "cpu",
				Usage: "limits the available cpu cores",
			},
			&cli.StringFlag{
				Name:  "memory",
				Usage: `limits the available memory (supports bytes but also e.g. "1MB" or "128 kilobytes")`,
			},
		},
	})
}

func run(ctx *cli.Context) (err error) {
	args := ctx.Args()
	if args.Len() == 0 {
		return fmt.Errorf("image not specified: use `foxbox run <image>`")
	}

	var v datasize.ByteSize
	if memory := ctx.String("memory"); memory != "" {
		err = v.UnmarshalText([]byte(memory))
		if err != nil {
			return fmt.Errorf("parsing memory flag: %w", err)
		}
	}

	image := args.First()

	id, err := client.Create(&client.CreateOptions{
		Image: image,
	})

	if err != nil {
		return err
	}

	if ctx.Bool("rm") {
		defer func() {
			err := client.Delete(id, nil)
			if err != nil {
				log.Printf("failed to delete foxbox %s: %s\n", id, err)
			}
		}()
	}

	err = client.Run(id, &client.RunOptions{
		Command:          args.Slice()[1:],
		EnableNetworking: true,
		MaxMemoryBytes:   uint(v.Bytes()),
		MaxCPUs:          float32(ctx.Float64("cpu")),
	})

	return
}
