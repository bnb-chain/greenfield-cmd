package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sptypes "github.com/bnb-chain/greenfield/x/sp/types"
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

	if len(spInfo) == 0 {
		return nil
	}

	var nameMaxLen int
	var endpointMaxLen int
	for _, info := range spInfo {
		lengthOfName := len(info.Description.GetMoniker())
		if lengthOfName > nameMaxLen {
			nameMaxLen = lengthOfName
		}
		if len(info.Endpoint) > endpointMaxLen {
			endpointMaxLen = len(info.Endpoint)
		}
	}

	format := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds  \n", nameMaxLen, operatorAddressLen, endpointMaxLen, len(exitStatus))

	fmt.Printf(format, "name", "operator address", "endpoint", "status")
	for _, info := range spInfo {
		fmt.Printf(format, info.Description.GetMoniker(), info.OperatorAddress, info.Endpoint, info.Status.String()[len(StatusSPrefix):])
	}

	return nil
}

func querySP(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return errors.New("the args should be one , please set the sp endpoint")
	}
	// sp addr could be an endpoint or sp operator address
	spAddressInfo := ctx.Args().Get(0)

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelQuerySP := context.WithCancel(globalContext)
	defer cancelQuerySP()

	addr, err := getSPAddr(spAddressInfo, client, c)
	if err != nil {
		return toCmdErr(err)
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
	spAddressInfo := ctx.Args().Get(0)

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelGetPrice := context.WithCancel(globalContext)
	defer cancelGetPrice()

	spAddr, err := getSPAddr(spAddressInfo, client, c)
	if err != nil {
		return err
	}

	price, err := client.GetStoragePrice(c, spAddr.String())
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
	fmt.Println("get bucket free quota:", price.FreeReadQuota)
	return nil
}

func getSPAddr(spAddressInfo string, cli client.Client, c context.Context) (sdk.AccAddress, error) {
	var addr sdk.AccAddress
	var err error
	var spList []sptypes.StorageProvider
	// the input sp info is operator address
	if len(spAddressInfo) == operatorAddressLen && strings.HasPrefix(spAddressInfo, "0x") {
		addr, err = sdk.AccAddressFromHexUnsafe(spAddressInfo)
		if err != nil {
			return nil, fmt.Errorf("the sp address %s is invalid", spAddressInfo)
		}
	} else {
		// the input sp info is a http endpoint
		spList, err = cli.ListStorageProviders(c, false)
		if err != nil {
			return nil, errors.New("fail to get SP info")
		}

		// if the command input the sp operator address
		var findSP bool
		for _, info := range spList {
			if info.Endpoint == spAddressInfo {
				addr = info.GetOperatorAccAddress()
				findSP = true
			}
		}
		if !findSP {
			return nil, errors.New("fail to get SP info, the input endpoint is invalid")
		}
	}
	return addr, nil
}
