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
			Name:    "keystore",
			Aliases: []string{"k"},
			Value:   defaultKeyfile,
			Usage:   "key file path",
		},
	}

	app := &cli.App{
		Name:  "gnfd-cmd",
		Usage: "cmd tool for supporting making request to greenfield",
		Flags: flags,
		Commands: []*cli.Command{
			{
				Name:  "bucket",
				Usage: "support the bucket operation functions, including create/update/delete/head/list",
				Subcommands: []*cli.Command{
					cmdCreateBucket(),
					cmdUpdateBucket(),
					cmdDelBucket(),
					cmdHeadBucket(),
					cmdListBuckets(),
				},
			},
			{
				Name:  "object",
				Usage: "support the object operation functions, including put/get/update/delete/head/list and so on",
				Subcommands: []*cli.Command{
					cmdPutObj(),
					cmdGetObj(),
					cmdDelObject(),
					cmdHeadObj(),
					cmdCancelObjects(),
					cmdListObjects(),
					cmdCalHash(),
					cmdCreateFolder(),
				},
			},
			{
				Name:  "group",
				Usage: "support the group operation functions, including create/update/delete/head/head-member",
				Subcommands: []*cli.Command{
					cmdCreateGroup(),
					cmdUpdateGroup(),
					cmdHeadGroup(),
					cmdHeadGroupMember(),
					cmdDelGroup(),
				},
			},
			{
				Name:  "crosschain",
				Usage: "support the cross-chain functions, including transfer and mirror",
				Subcommands: []*cli.Command{
					cmdMirrorResource(),
					cmdTransferOut(),
				},
			},
			{
				Name:  "bank",
				Usage: "support the bank functions, including transfer and get balance",
				Subcommands: []*cli.Command{
					cmdTransfer(),
					cmdGetAccountBalance(),
				},
			},
			{
				Name:  "permission",
				Usage: "support object policy and bucket policy operation functions",
				Subcommands: []*cli.Command{
					cmdPutObjPolicy(),
					cmdPutBucketPolicy(),
				},
			},

			{
				Name:  "payment",
				Usage: "support the payment operation functions",
				Subcommands: []*cli.Command{
					cmdCreatePaymentAccount(),
					cmdPaymentDeposit(),
					cmdPaymentWithdraw(),
					cmdListPaymentAccounts(),
					cmdBuyQuota(),
					cmdGetQuotaPrice(),
					cmdGetQuotaInfo(),
				},
			},
			{
				Name:  "sp",
				Usage: "support the storage provider operation functions",
				Subcommands: []*cli.Command{
					cmdListSP(),
					cmdGetSP(),
				},
			},

			cmdGenerateKey(),
		},
	}
	app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config"))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
