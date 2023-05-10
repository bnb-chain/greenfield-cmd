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
				Name:  "host",
				Usage: "host name of request",
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:  "rpcAddr",
				Usage: "greenfield chain client rpc address",
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
				Name:     passwordFileFlag,
				Usage:    "password file for encrypting and decoding the private key",
				Required: false,
			},
		),
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   "config.toml",
			Usage:   "Load configuration from `FILE`",
		},
		&cli.StringFlag{
			Name:        "keystore",
			Aliases:     []string{"k"},
			DefaultText: defaultKeyfile,
			Usage:       "key file path",
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
			cmdPutObjPolicy(),
			cmdPutBucketPolicy(),
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
			cmdTransfer(),
			cmdTransferOut(),
			cmdCreatePaymentAccount(),
			cmdPaymentDeposit(),
			cmdPaymentWithdraw(),
			cmdListPaymentAccounts(),
			cmdGetAccountBalance(),
			cmdMirrorResource(),
			cmdGenerateKey(),
		},
	}
	app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config"))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
