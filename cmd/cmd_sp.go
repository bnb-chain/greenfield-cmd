package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

func cmdListSP() *cli.Command {
	return &cli.Command{
		Name:      "ls-sp",
		Action:    ListSP,
		Usage:     "list storage providers info",
		ArgsUsage: "",
		Description: `
List the storage provider info including the endpoint and the address on chain

Examples:
$ gnfd-cmd ls-sp `,
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
