package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/bnb-chain/greenfield-go-sdk/types"
	gtypes "github.com/bnb-chain/greenfield/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
)

// cmdSetTag Set tag for a given existing resource GRN (a bucket, a object or a group)
func cmdSetTag() *cli.Command {
	return &cli.Command{
		Name:      "set",
		Action:    setTag,
		Usage:     "Set tag for a given existing resource GRN",
		ArgsUsage: " RESOURCE-URL",
		Description: `
The command is used to set tag for a given existing resource GRN (a bucket, a object or a group).

the resource url can be the follow types:
1) grn:b::bucketname, is the GRN of the bucket "bucketname"
2) grn:o::bucketname/objectname, is the GRN of object "gnfd://bucketname/objectname"
3) grn:g:owneraddress:groupname, is the GRN of group "groupname"

Examples:
$ gnfd-cmd tag set --tags='[{"key":"key1","value":"value1"},{"key":"key2","value":"value2"}]' grn:o::gnfd-bucket/gnfd-object`,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  tagFlag,
				Value: "",
				Usage: "set one or more tags for the given GRN. The tag value is key-value pairs in json array format. E.g. [{\"key\":\"key1\",\"value\":\"value1\"},{\"key\":\"key2\",\"value\":\"value2\"}]",
			},
		},
	}
}

// setTag Set tag for a given existing resource GRN (a bucket, a object or a group)
func setTag(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	grnParam := ctx.Args().Get(0)
	var grn gtypes.GRN
	parsingGrnErr := grn.ParseFromString(grnParam, true)
	if parsingGrnErr != nil {
		return toCmdErr(parsingGrnErr)
	}

	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	tagsParam := ctx.String(tagFlag)
	if tagsParam == "" {
		toCmdErr(errors.New("invalid tags parameter"))
	}
	tags := &storageTypes.ResourceTags{}
	err = json.Unmarshal([]byte(tagsParam), &tags.Tags)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()
	txnHash, err := client.SetTag(c, grnParam, *tags, types.SetTagsOptions{})

	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txnHash, "SetTags")
	if err != nil {
		return toCmdErr(err)
	}

	return nil
}
