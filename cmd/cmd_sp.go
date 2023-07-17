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
		ArgsUsage: "<Storage Provider endpoint>",
		Description: `
Get the storage provider details information, including status, address and so on.

Examples:
$ gnfd-cmd sp head https://gnfd-testnet-sp-1.nodereal.io`,
	}
}

// cmdGetQuotaPrice query the quota price of the specific sp
func cmdGetQuotaPrice() *cli.Command {
	return &cli.Command{
		Name:      "get-price",
		Action:    getQuotaPrice,
		Usage:     "get the quota price of the SP",
		ArgsUsage: "<Storage Provider endpoint>",
		Description: `
Get the quota price and the storage price of the specific Storage Provider.

Examples:
$ gnfd-cmd sp get-price https://gnfd-testnet-sp-1.nodereal.io`,
	}
}

func ListSP(ctx *cli.Context) error {
	client, err := NewClient(ctx, true)
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
	if ctx.NArg() != 1 {
		return errors.New("the args should be one , please set the sp endpoint")
	}
	endpoint := ctx.Args().Get(0)

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spList, err := client.ListStorageProviders(c, false)
	if err != nil {
		return toCmdErr(errors.New("fail to get SP info"))
	}

	var addr sdk.AccAddress
	var findSP bool
	for _, info := range spList {
		if info.Endpoint == endpoint {
			addr = info.GetOperatorAccAddress()
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
	fmt.Println("Status:", spInfo.Status)
	return nil
}

// getQuotaPrice query the quota price info of sp from greenfield chain
func getQuotaPrice(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return errors.New("the args should be one , please set the sp endpoint")
	}
	endpoint := ctx.Args().Get(0)

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spList, err := client.ListStorageProviders(c, false)
	if err != nil {
		return toCmdErr(errors.New("fail to get SP info"))
	}

	var spAddressStr string
	var findSP bool
	for _, info := range spList {
		if info.Endpoint == endpoint {
			spAddressStr = info.GetOperatorAddress()
			findSP = true
		}
	}

	if !findSP {
		return toCmdErr(errors.New("fail to get SP info"))
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
