package cmd

import (
	"github.com/urfave/cli/v3"
)

// Version is set via ldflags at build time.
var Version = "dev"

// NewApp creates the root dctl CLI command.
func NewApp() *cli.Command {
	return &cli.Command{
		Name:    "dctl",
		Usage:   "Docker Compose compatible CLI for Apple container",
		Version: Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug output",
				Sources: cli.EnvVars("DCTL_DEBUG"),
			},
		},
		Commands: composeCommands(),
	}
}
