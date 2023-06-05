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

		&cli.StringFlag{
			Name:    passwordFileFlag,
			Aliases: []string{"p"},
			Usage:   "password file for encrypting and decoding the private key",
		},

		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load configuration from `FILE`",
		},
		&cli.StringFlag{
			Name:    "keystore",
			Aliases: []string{"k"},
			Usage:   "keystore file path",
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
					cmdBuyQuota(),
					cmdGetQuotaInfo(),
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
					cmdUpdateObject(),
					cmdGetUploadProgress(),
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
				Name:  "bank",
				Usage: "support the bank functions, including transfer in greenfield and query balance",
				Subcommands: []*cli.Command{
					cmdTransfer(),
					cmdGetAccountBalance(),
					cmdTransferOut(),
				},
			},
			{
				Name:  "policy",
				Usage: "support object,bucket and group policy operation functions",
				Subcommands: []*cli.Command{
					cmdPutPolicy(),
					cmdDelPolicy(),
				},
			},

			{
				Name:  "payment-account",
				Usage: "support the payment account operation functions",
				Subcommands: []*cli.Command{
					cmdCreatePaymentAccount(),
					cmdPaymentDeposit(),
					cmdPaymentWithdraw(),
					cmdListPaymentAccounts(),
				},
			},
			{
				Name:  "sp",
				Usage: "support the storage provider operation functions",
				Subcommands: []*cli.Command{
					cmdListSP(),
					cmdGetSP(),
					cmdGetQuotaPrice(),
				},
			},

			{
				Name:  "keystore",
				Usage: "support the keystore operation functions",
				Subcommands: []*cli.Command{
					cmdGenerateKey(),
					cmdPrintKey(),
				},
			},
		},
	}
	app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config"))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
