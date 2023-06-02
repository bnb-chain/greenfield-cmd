package main

import (
	"context"
	"cosmossdk.io/math"
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
		ArgsUsage: "GROUP-NAME",
		Description: `
Create a new group

Examples:
$ gnfd-cmd group make-group group-name`,
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
		ArgsUsage: "GROUP-NAME",
		Description: `
Add or remove group members of the group, you can set add members 
and remove members list at the same time.
You need also set group owner using --groupOwner if you are not the owner of the group.

Examples:
$ gnfd-cmd group update-group --groupOwner 0x.. --addMembers 0x.. group-name`,
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

func cmdMirrorGroup() *cli.Command {
	return &cli.Command{
		Name:      "mirror",
		Action:    mirrorGroup,
		Usage:     "mirror group to BSC",
		ArgsUsage: "",
		Description: `
Mirror a group as NFT to BSC

Examples:
# Mirror a group using group id
$ gnfd-cmd group mirror --id 1

# Mirror a group using group name
$ gnfd-cmd group mirror --name yourGroupName
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     IdFlag,
				Value:    "",
				Usage:    "group id",
				Required: false,
			},
			&cli.StringFlag{
				Name:     groupNameFlag,
				Value:    "",
				Usage:    "group name",
				Required: false,
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

	err = waitTxnStatus(client, c, txnHash, "CreateGroup")
	if err != nil {
		return toCmdErr(err)
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

	err = waitTxnStatus(client, c, txnHash, "UpdateGroupMember")
	if err != nil {
		return toCmdErr(err)
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

func mirrorGroup(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	id := math.NewUintFromString(ctx.String(IdFlag))
	groupName := ctx.String(groupNameFlag)

	c, cancelContext := context.WithCancel(globalContext)
	defer cancelContext()

	txResp, err := client.MirrorGroup(c, id, groupName, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("mirror group succ, txHash: %s\n", txResp.TxHash)
	return nil
}
