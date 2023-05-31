package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdCreateBucket create a new Bucket
func cmdCreateGroup() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Action:    createGroup,
		Usage:     "create a new group",
		ArgsUsage: "GROUP-URL",
		Description: `
Create a new group, the group name need to set by GROUP-URL like "gnfd://groupName"

Examples:
$ gnfd-cmd group make-group gnfd://group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  initMemberFlag,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
		},
	}
}

// cmdUpdateGroup add or delete group member to the group
func cmdUpdateGroup() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Action:    updateGroupMember,
		Usage:     "update group member",
		ArgsUsage: "GROUP-URL",
		Description: `
Add or remove group members of the group, you can set add members 
and remove members list at the same time.
You need also set group owner using --groupOwner if you are not the owner of the group.

Examples:
$ gnfd-cmd group update-group --groupOwner 0x.. --addMembers 0x.. gnfd://group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  addMemberFlag,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  removeMemberFlag,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  groupOwnerFlag,
				Value: "",
				Usage: "need set the owner address if you are not the owner of the group",
			},
		},
	}
}

// createGroup send the create bucket request to storage provider
func createGroup(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	opts := sdktypes.CreateGroupOptions{}

	initMembersInfo := ctx.String(initMemberFlag)
	// set group init members if provided by user
	if initMembersInfo != "" {
		addrList, err := parseAddrList(initMembersInfo)
		if err != nil {
			return toCmdErr(err)
		}
		opts.InitGroupMember = addrList
	}

	opts.TxOpts = &types.TxOption{Mode: &SyncBroadcastMode}

	c, cancelCreateGroup := context.WithCancel(globalContext)
	defer cancelCreateGroup()

	txnHash, err := client.CreateGroup(c, groupName, opts)
	if err != nil {
		return toCmdErr(err)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), ContextTimeout)
	defer cancel()

	txnResponse, err := client.WaitForTx(ctxTimeout, txnHash)
	if err != nil {
		return toCmdErr(fmt.Errorf("the txn: %s ,has been submitted, please check it later:%v", txnHash, err))
	}
	if txnResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("the createGroup txn: %s has failed with response code: %d", txnHash, txnResponse.Code))
	}

	groupOwner, err := getGroupOwner(ctx, client)
	if err == nil {
		info, err := client.HeadGroup(c, groupName, groupOwner)
		if err == nil {
			fmt.Printf("create group: %s succ, txn hash:%s, group id: %s \n", groupName, txnHash, info.Id.String())
			return nil
		}
	}

	fmt.Printf("create group: %s succ, txn hash:%s \n", groupName, txnHash)

	return nil
}

func updateGroupMember(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	addMembersInfo := ctx.String(addMemberFlag)
	removeMembersInfo := ctx.String(removeMemberFlag)

	if addMembersInfo == "" && removeMembersInfo == "" {
		return toCmdErr(errors.New("fail to get members to update"))
	}

	var addGroupMembers []string
	var removeGroupMembers []string
	if strings.Contains(addMembersInfo, ",") {
		addGroupMembers = strings.Split(addMembersInfo, ",")
	} else if addMembersInfo != "" {
		addGroupMembers = []string{addMembersInfo}
	}

	if strings.Contains(removeMembersInfo, ",") {
		removeGroupMembers = strings.Split(removeMembersInfo, ",")
	} else if removeMembersInfo != "" {
		removeGroupMembers = []string{removeMembersInfo}
	}

	groupOwner, err := getGroupOwner(ctx, client)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelUpdateGroup := context.WithCancel(globalContext)
	defer cancelUpdateGroup()

	_, err = client.HeadGroup(c, groupName, groupOwner)
	if err != nil {
		return toCmdErr(ErrGroupNotExist)
	}

	txOpts := &types.TxOption{Mode: &SyncBroadcastMode}
	txnHash, err := client.UpdateGroupMember(c, groupName, groupOwner, addGroupMembers, removeGroupMembers,
		sdktypes.UpdateGroupMemberOption{TxOpts: txOpts})
	if err != nil {
		return toCmdErr(err)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), ContextTimeout)
	defer cancel()

	txnResponse, err := client.WaitForTx(ctxTimeout, txnHash)
	if err != nil {
		return toCmdErr(fmt.Errorf("the txn: %s ,has been submitted, please check it later:%v", txnHash, err))
	}
	if txnResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("the updateMember txn: %s has failed with response code: %d", txnHash, txnResponse.Code))
	}
	fmt.Printf("update group: %s succ, txn hash:%s \n", groupName, txnHash)
	return nil
}

func getGroupOwner(ctx *cli.Context, client client.Client) (string, error) {
	groupOwnerAddrStr := ctx.String(groupOwnerFlag)

	if groupOwnerAddrStr != "" {
		return groupOwnerAddrStr, nil
	}

	acc, err := client.GetDefaultAccount()
	if err != nil {
		return "", toCmdErr(err)
	}

	return acc.GetAddress().String(), nil
}
