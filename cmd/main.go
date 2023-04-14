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
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load configuration from `FILE`",
		},
	}

	app := &cli.App{
		Name:  "gnfd-cmd",
		Usage: "cmd tool for supporting making request to greenfield",
		Flags: flags,
		Commands: []*cli.Command{
			cmdCreateBucket(),
			cmdUpdateBucket(),
			cmdPutObj(),
			cmdGetObj(),
			cmdCreateObj(),
			cmdCancelObjects(),
			cmdCalHash(),
			cmdDelObject(),
			cmdDelBucket(),
			cmdHeadObj(),
			cmdHeadBucket(),
			cmdListSP(),
			cmdCreateGroup(),
			cmdUpdateGroup(),
			cmdHeadGroup(),
			cmdHeadGroupMember(),
			cmdDelGroup(),
			cmdBuyQuota(),
			cmdGetQuotaPrice(),
			cmdGetQuotaInfo(),
			cmdListBuckets(),
			cmdListObjects(),
		},
	}
	app.Before = func(ctx *cli.Context) error {
		BeforeFunc := altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config"))
		_ = BeforeFunc(ctx)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
