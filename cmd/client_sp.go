package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

func cmdListSP() *cli.Command {
	return &cli.Command{
		Name:      "list-sp",
		Action:    ListSP,
		Usage:     "list sp info",
		ArgsUsage: "",
		Description: `

Examples:
$ gnfd-cmd  list-sp `,
	}
}

func ListSP(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spInfo, err := client.ListSP(c)
	if err != nil {
		fmt.Println("fail to list sp:", err.Error())
		return err
	}

	fmt.Println("sp list:")
	for id, info := range spInfo {
		fmt.Println(fmt.Sprintf("sp[%d]: operator-address:%s, endpoint:%s,"+
			" Status:%s", id, info.OperatorAddress, info.Endpoint, info.Status))
	}
	return nil
}
