package cli

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

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
			&cli.UintFlag{
				Name:  "max-pids",
				Usage: "limits the number of processes",
			},
			&cli.StringSliceFlag{
				Name:    "volume",
				Aliases: []string{"v"},
				Usage:   "mounts local volumes in the format host:box",
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

	var volumes []client.VolumeConfig
	if v := ctx.StringSlice("volume"); len(v) > 0 {
		for _, v := range v {
			parts := strings.Split(v, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid volume config (%v): must be formatted host:box", v)
			}
			volumes = append(volumes, client.VolumeConfig{
				HostPath: parts[0],
				BoxPath:  parts[1],
			})
		}
	}

	id, err := foxbox.Create(&client.CreateOptions{
		Image: args.First(),
	})

	if err != nil {
		return err
	}

	if ctx.Bool("rm") {
		defer func() {
			err := foxbox.Delete(id, nil)
			if err != nil {
				log.Printf("failed to delete foxbox %s: %s\n", id, err)
			}
		}()
	}

	err = foxbox.Run(id, &client.RunOptions{
		Command:          args.Slice()[1:],
		EnableNetworking: true,
		MaxMemoryBytes:   uint(v.Bytes()),
		MaxCPUs:          float32(ctx.Float64("cpu")),
		MaxProcesses:     ctx.Uint("max-pids"),
		Volumes:          volumes,
	})

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		os.Exit(exitError.ExitCode())
	}

	return
}
