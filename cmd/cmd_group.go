package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sdkTypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdCreateBucket create a new Bucket
func cmdCreateGroup() *cli.Command {
	return &cli.Command{
		Name:      "make-group",
		Action:    createGroup,
		Usage:     "create a new group",
		ArgsUsage: "GROUP-URL",
		Description: `
Create a new group, the group name need to set by GROUP-URL like "gnfd://groupName"

Examples:
$ gnfd-cmd -c config.toml make-group gnfd://group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  initMemberFlagName,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
		},
	}
}

// cmdUpdateGroup add or delete group member to the group
func cmdUpdateGroup() *cli.Command {
	return &cli.Command{
		Name:      "update-group",
		Action:    updateGroupMember,
		Usage:     "update group member",
		ArgsUsage: "GROUP-URL",
		Description: `
Add or remove group members of the group, you can set add members 
and remove members list at the same time.
You need also set group owner using --groupOwner if you are not the owner of the group.

Examples:
$ gnfd-cmd -c config.toml update-group --groupOwner 0x.. --addMembers 0x.. gnfd://group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  addMemberFlagName,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  removeMemberFlagName,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  groupOwnerFlagName,
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

	opts := sdkTypes.CreateGroupOptions{}

	initMembersInfo := ctx.String(initMemberFlagName)
	// set group init members if provided by user
	if initMembersInfo != "" {
		addrList, err := parseAddrList(initMembersInfo)
		if err != nil {
			return toCmdErr(err)
		}
		opts.InitGroupMember = addrList
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	opts.TxOpts = &types.TxOption{Mode: &broadcastMode}

	c, cancelCreateGroup := context.WithCancel(globalContext)
	defer cancelCreateGroup()

	txnHash, err := client.CreateGroup(c, groupName, opts)
	if err != nil {
		return toCmdErr(err)
	}
	c, cancelGroup := context.WithCancel(globalContext)
	defer cancelGroup()

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

	addMembersInfo := ctx.String(addMemberFlagName)
	removeMembersInfo := ctx.String(removeMemberFlagName)

	if addMembersInfo == "" && removeMembersInfo == "" {
		return toCmdErr(errors.New("fail to get members to update"))
	}

	addGroupMembers := strings.Split(addMembersInfo, ",")
	removeGroupMembers := strings.Split(removeMembersInfo, ",")

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

	txnHash, err := client.UpdateGroupMember(c, groupName, groupOwner, addGroupMembers, removeGroupMembers, sdkTypes.UpdateGroupMemberOption{})
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("update group: %s succ, txn hash:%s \n", groupName, txnHash)
	return nil
}

func getGroupOwner(ctx *cli.Context, client client.Client) (string, error) {
	groupOwnerAddrStr := ctx.String(groupOwnerFlagName)

	if groupOwnerAddrStr != "" {
		return groupOwnerAddrStr, nil
	} else {
		acc, err := client.GetDefaultAccount()
		if err != nil {
			return "", toCmdErr(err)
		}
		return acc.GetAddress().String(), nil
	}
	return "", errors.New("fail to fetch group owner")
}
