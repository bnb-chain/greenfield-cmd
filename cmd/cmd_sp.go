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
		Usage:     "list sp info",
		ArgsUsage: "",
		Description: `

Examples:
$ gnfd-cmd  ls-sp `,
	}
}

func ListSP(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spInfo, err := client.ListSP(c, false)
	if err != nil {
		fmt.Println("fail to list sp:", err.Error())
		return nil
	}

	fmt.Println("sp list:")
	for id, info := range spInfo {
		fmt.Println(fmt.Sprintf("sp[%d]: operator-address:%s, endpoint:%s,"+
			" Status:%s", id, info.OperatorAddress, info.Endpoint, info.Status))
	}
	return nil
}
