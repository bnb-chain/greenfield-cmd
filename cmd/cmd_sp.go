package main

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

func cmdListSP() *cli.Command {
	return &cli.Command{
		Name:      "ls",
		Action:    ListSP,
		Usage:     "list storage providers info",
		ArgsUsage: "",
		Description: `
List the storage provider info including the endpoint and the address on chain

Examples:
$ gnfd-cmd sp ls `,
	}
}

func cmdGetSP() *cli.Command {
	return &cli.Command{
		Name:      "head",
		Action:    querySP,
		Usage:     "get storage provider details",
		ArgsUsage: "",
		Description: `
Let the storage provider details information , including status, address and so on

Examples:
$ gnfd-cmd sp head --spEndpoint https://...`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     spEndpointFlag,
				Value:    "",
				Usage:    "indicate the storage provider chain address string",
				Required: true,
			},
		},
	}
}

// cmdGetQuotaPrice query the quota price of the specific sp
func cmdGetQuotaPrice() *cli.Command {
	return &cli.Command{
		Name:      "get-price",
		Action:    getQuotaPrice,
		Usage:     "get the quota price of the SP",
		ArgsUsage: "",
		Description: `
Get the quota price of the specific sp, the command need to set the sp address with --spAddress
The command need to set the SP info with --spAddress.

Examples:
$ gnfd-cmd payment get-price --spAddress "0x.."`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     spAddressFlag,
				Value:    "",
				Usage:    "indicate the storage provider chain address string",
				Required: true,
			},
		},
	}
}

func ListSP(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spInfo, err := client.ListStorageProviders(c, false)
	if err != nil {
		fmt.Println("fail to list SP:", err.Error())
		return nil
	}

	fmt.Println("SP list:")
	for id, info := range spInfo {
		fmt.Println(fmt.Sprintf("sp[%d]: operator-address:%s, endpoint:%s,"+
			" Status:%s", id, info.OperatorAddress, info.Endpoint, info.Status))
	}
	return nil
}

func querySP(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	endpoint := ctx.String(spEndpointFlag)
	if endpoint == "" {
		return toCmdErr(errors.New("fail to fetch sp endpoint"))
	}

	spList, err := client.ListStorageProviders(c, false)
	if err != nil {
		return toCmdErr(errors.New("fail to get SP info"))
	}

	var addr sdk.AccAddress
	var findSP bool
	for _, info := range spList {
		if info.Endpoint == endpoint {
			addr = info.GetOperator()
			findSP = true
		}
	}

	if !findSP {
		return toCmdErr(errors.New("fail to get SP info"))
	}

	spInfo, err := client.GetStorageProviderInfo(c, addr)
	if err != nil {
		return toCmdErr(errors.New("fail to get SP info"))
	}

	fmt.Println("SP info:")
	fmt.Println(spInfo.String())
	return nil
}

// getQuotaPrice query the quota price info of sp from greenfield chain
func getQuotaPrice(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spAddressStr := ctx.String(spAddressFlag)
	if spAddressStr == "" {
		return toCmdErr(errors.New("fail to fetch sp address"))
	}

	price, err := client.GetStoragePrice(c, spAddressStr)
	if err != nil {
		return toCmdErr(err)
	}

	quotaPrice, err := price.ReadPrice.Float64()
	if err != nil {
		fmt.Println("get quota price error:", err.Error())
		return err
	}

	storagePrice, err := price.StorePrice.Float64()
	if err != nil {
		fmt.Println("get storage price error:", err.Error())
		return err
	}

	fmt.Println("get bucket read quota price:", quotaPrice, " wei/byte")
	fmt.Println("get bucket storage price:", storagePrice, " wei/byte")
	return nil
}
