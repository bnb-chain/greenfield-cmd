package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdHeadObj query the object info and return
func cmdHeadObj() *cli.Command {
	return &cli.Command{
		Name:      "head-obj",
		Action:    headObject,
		Usage:     "query object info",
		ArgsUsage: "OBJECT-URL",
		Description: `
send headObject txn to chain and fetch objectInfo on greenfield chain
Examples:
$ gnfd-cmd head-bucket gnfd://bucket-name/object-name`,
	}
}

// cmdHeadBucket query the bucket info and return
func cmdHeadBucket() *cli.Command {
	return &cli.Command{
		Name:      "head-bucket",
		Action:    headBucket,
		Usage:     "query bucket info",
		ArgsUsage: "BUCKET-URL",
		Description: `
send headBucket txn to chain and fetch bucketInfo on greenfield chain
Examples:
$ gnfd-cmd head-bucket gnfd://bucket-name`,
	}
}

// cmdHeadGroup query the group info and return
func cmdHeadGroup() *cli.Command {
	return &cli.Command{
		Name:      "head-group",
		Action:    headGroup,
		Usage:     "query group info",
		ArgsUsage: "GROUP-URL",
		Description: `
send headGroup txn to chain and fetch bucketInfo on greenfield chain
Examples:
$ gnfd-cmd head-group --groupOwner  gnfd://group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  groupOwnerFlagName,
				Value: "",
				Usage: "need set the owner address if you are not the owner of the group",
			},
		},
	}
}

// cmdHeadGroupMember query the group member if it exists in group
func cmdHeadGroupMember() *cli.Command {
	return &cli.Command{
		Name:      "head-member",
		Action:    headGroupMember,
		Usage:     "check group member if it exists",
		ArgsUsage: "GROUP-URL",
		Description: `
send headGroupMember txn to chain and check if member is in the group
Examples:
$ gnfd-cmd  head-group --headMember  gnfd://group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  groupOwnerFlagName,
				Value: "",
				Usage: "need set the owner address if you are not the owner of the group",
			},
			&cli.StringFlag{
				Name:  headMemberFlagName,
				Value: "",
				Usage: "indicate the head member address",
			},
		},
	}
}

func headObject(ctx *cli.Context) error {
	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadObject := context.WithCancel(globalContext)
	defer cancelHeadObject()

	objectInfo, err := client.HeadObject(c, bucketName, objectName)
	if err != nil {
		fmt.Println("no such object")
		return nil
	}
	parseChainInfo(objectInfo.String(), false)
	return nil
}

func headBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadBucket := context.WithCancel(globalContext)
	defer cancelHeadBucket()

	bucketInfo, err := client.HeadBucket(c, bucketName)
	if err != nil {
		fmt.Println("no such bucket")
		return nil
	}

	parseChainInfo(bucketInfo.String(), true)
	return nil
}

func headGroup(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadGroup := context.WithCancel(globalContext)
	defer cancelHeadGroup()

	groupOwner, err := getGroupOwner(ctx, client)
	if err != nil {
		return toCmdErr(err)
	}

	groupInfo, err := client.HeadGroup(c, groupName, groupOwner)
	if err != nil {
		fmt.Println("no such group")
		return nil
	}

	infoStr := strings.Split(groupInfo.String(), " ")
	for _, info := range infoStr {
		fmt.Println(info)
	}
	return nil
}

func headGroupMember(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadBucket := context.WithCancel(globalContext)
	defer cancelHeadBucket()

	groupOwner, err := getGroupOwner(ctx, client)
	if err != nil {
		return toCmdErr(err)
	}

	headMember := ctx.String(headMemberFlagName)
	if headMember == "" {
		return toCmdErr(errors.New("no head member address"))
	}
	headMemberAddr, err := sdk.AccAddressFromHexUnsafe(headMember)
	if err != nil {
		return toCmdErr(err)
	}

	exist := client.HeadGroupMember(c, groupName, groupOwner, headMemberAddr)
	if !exist {
		fmt.Println("the user does not exist in the group")
		return nil
	}

	fmt.Println("the user is a member of the group")
	return nil
}
