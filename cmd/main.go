package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var globalContext, globalCancel = context.WithCancel(context.Background())

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Usage:       "Load configuration from config.toml",
			DefaultText: "./config.toml",
			Value:       "config.toml",
		},
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:  "endpoint",
				Usage: "sp provider endpoint info",
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:  "host",
				Usage: "primary host",
			},
		),
	}

	app := &cli.App{
		Name:  "gnfd",
		Usage: "client for inscription blockchain object storage",
		Flags: flags,
		Commands: []*cli.Command{
			cmdMakeBucket(),
			cmdSendPutTxn(),
			cmdPutObj(),
			cmdGetObj(),
			cmdPreCreateObj(),
			cmdPreMakeBucket(),
			cmdCalHash(),
		},
		Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config")),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
