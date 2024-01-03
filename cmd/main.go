package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "net/http/pprof"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// var cpuprofile = "/Users/ruojunm/binance_workspace/greenfield-cmd/build/profiling/cpu.prof"
// var memprofile = "/Users/ruojunm/binance_workspace/greenfield-cmd/build/profiling/memprofile.prof"
var globalContext, _ = context.WithCancel(context.Background())

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

var args = os.Args

func main() {
	//这里实现了远程获取pprof数据的接口
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()
	mainWithArg(os.Args)
}
func mainWithArg(args []string) {
	fmt.Println(args)
	//if cpuprofile != "" {
	//	f, err := os.Create(cpuprofile)
	//	if err != nil {
	//		log.Fatal("could not create CPU profile: ", err)
	//	}
	//	defer f.Close() // error handling omitted for example
	//	if err := pprof.StartCPUProfile(f); err != nil {
	//		log.Fatal("could not start CPU profile: ", err)
	//	}
	//	defer pprof.StopCPUProfile()
	//}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir, err = os.Getwd()
		if err != nil {
			fmt.Println("fail to get home dir or local dir")
		}
	}

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
			Name:    configFlag,
			Aliases: []string{"c"},
			Usage:   "Load configuration from `FILE`",
		},
		&cli.StringFlag{
			Name:    keyStoreFlag,
			Aliases: []string{"k"},
			Usage:   "keystore file path",
		},
		&cli.StringFlag{
			Name:  homeFlag,
			Usage: "directory for config and keystore",
			Value: filepath.Join(homeDir, DefaultConfigDir),
		},
	}

	app := &cli.App{
		Name:  "gnfd-cmd",
		Usage: "cmd tool for supporting making request to greenfield",
		Flags: flags,
		Commands: []*cli.Command{
			{
				Name:  "bucket",
				Usage: "support the bucket operation functions, including create/update/delete/head/list and so on",
				Subcommands: []*cli.Command{
					cmdCreateBucket(),
					cmdUpdateBucket(),
					cmdDelBucket(),
					cmdHeadBucket(),
					cmdListBuckets(),
					cmdBuyQuota(),
					cmdGetQuotaInfo(),
					cmdMirrorBucket(),
					cmdSetTagForBucket(),
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
					cmdUpdateObject(),
					cmdGetUploadProgress(),
					cmdMirrorObject(),
					cmdSetTagForObject(),
				},
			},
			{
				Name:  "group",
				Usage: "support the group operation functions, including create/update/delete/head/head-member/mirror/renew",
				Subcommands: []*cli.Command{
					cmdCreateGroup(),
					cmdUpdateGroup(),
					cmdHeadGroup(),
					cmdHeadGroupMember(),
					cmdDelGroup(),
					cmdMirrorGroup(),
					cmdRenewGroup(),
					cmdListGroup(),
					cmdListGroupMember(),
					cmdListGroupBelong(),
					cmdSetTagForGroup(),
				},
			},

			{
				Name:  "bank",
				Usage: "support the bank functions, including transfer in greenfield and query balance",
				Subcommands: []*cli.Command{
					cmdTransfer(),
					cmdGetAccountBalance(),
					cmdBridge(),
				},
			},
			{
				Name:  "policy",
				Usage: "support object,bucket and group policy operation functions",
				Subcommands: []*cli.Command{
					cmdPutPolicy(),
					cmdDelPolicy(),
					cmdListPolicy(),
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
				Name:  "account",
				Usage: "support the keystore operation functions",
				Subcommands: []*cli.Command{
					cmdImportAccount(),
					cmdListAccount(),
					cmdCreateAccount(),
					cmdExportAccount(),
					cmdSetDefaultAccount(),
				},
			},
			cmdShowVersion(),
		},
	}
	app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config"))

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	//if memprofile != "" {
	//	f, err := os.Create(memprofile)
	//	if err != nil {
	//		log.Fatal("could not create memory profile: ", err)
	//	}
	//	defer f.Close() // error handling omitted for example
	//	runtime.GC()    // get up-to-date statistics
	//	if err := pprof.WriteHeapProfile(f); err != nil {
	//		log.Fatal("could not write memory profile: ", err)
	//	}
	//}

}
