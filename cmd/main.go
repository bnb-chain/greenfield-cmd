package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var globalContext, _ = context.WithCancel(context.Background())

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
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:  "grpcAddr",
				Usage: "greenfield chain client grpc adress",
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:  "chainId",
				Usage: "greenfield chainId",
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "privateKey",
				Usage:    "hex encoding private key string",
				Required: false,
			},
		),
	}

	app := &cli.App{
		Name:  "gnfd-cmd",
		Usage: "cmd tool for supporting greenfield storage functions",
		Flags: flags,
		Commands: []*cli.Command{
			cmdCreateBucket(),
			cmdPutObj(),
			cmdGetObj(),
			cmdPreCreateObj(),
			cmdCalHash(),
			cmdDelObject(),
			cmdDelBucket(),
			cmdHeadObj(),
			cmdHeadBucket(),
			cmdChallenge(),
			cmdListSP(),
		},
		Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config")),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
