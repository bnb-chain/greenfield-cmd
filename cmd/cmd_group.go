package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
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
$ gnfd-cmd group create group-name`,
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
$ gnfd-cmd group update --groupOwner 0x.. --addMembers 0x.. group-name`,
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
			&cli.Int64Flag{
				Name:     groupMemberExpireFlag,
				Value:    0,
				Usage:    "set the expire timestamp for the addMember, it will apply to all the add members",
				Required: false,
			},
		},
	}
}

// cmdUpdateGroup add or delete group member to the group
func cmdRenewGroup() *cli.Command {
	return &cli.Command{
		Name:      "renew",
		Action:    renewGroupMember,
		Usage:     "update the expire time of group member",
		ArgsUsage: "GROUP-NAME",
		Description: `
renew expiration time of a list of group members 
You need also set group owner using --groupOwner if you are not the owner of the group.

Examples:
$ gnfd-cmd group renew --groupOwner 0x.. --renewMembers 0x..  --expireTime 1691569957 group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  renewMemberFlag,
				Value: "",
				Usage: "indicate the init member addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  groupOwnerFlag,
				Value: "",
				Usage: "need set the owner address if you are not the owner of the group",
			},
			&cli.Int64Flag{
				Name:  groupMemberExpireFlag,
				Value: 0,
				Usage: "set the expire timestamp for the addMember, it will apply to all the add members",
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
$ gnfd-cmd group mirror --groupName yourGroupName
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

	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	opts := sdktypes.CreateGroupOptions{}

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
	groupOwner, err := getGroupOwner(ctx)
	if err == nil {
		info, err := client.HeadGroup(c, groupName, groupOwner)
		if err == nil {
			fmt.Printf("make_group: %s \ntransaction hash: %s\ngroup id: %s \n",
				groupName, txnHash, info.Id.String())
			return nil
		}
	}

	fmt.Printf("make_group: %s \ntransaction hash: %s\n", groupName, txnHash)
	return nil
}

func updateGroupMember(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, false)
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

	groupOwner, err := getGroupOwner(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	expireTimestamp := ctx.Int64(expireTimeFlag)
	// set default expire timestamp
	if expireTimestamp == 0 {
		expireTimestamp = storageTypes.MaxTimeStamp.Unix()
	}

	addMemberNum := len(addGroupMembers)
	expireTimeList := make([]time.Time, addMemberNum)
	for i := 0; i < addMemberNum; i++ {
		expireTimeList[i] = time.Unix(expireTimestamp, 0)
	}

	c, cancelUpdateGroup := context.WithCancel(globalContext)
	defer cancelUpdateGroup()

	_, err = client.HeadGroup(c, groupName, groupOwner)
	if err != nil {
		return toCmdErr(ErrGroupNotExist)
	}

	txOpts := &types.TxOption{Mode: &SyncBroadcastMode}
	txnHash, err := client.UpdateGroupMember(c, groupName, groupOwner, addGroupMembers, removeGroupMembers, expireTimeList,
		sdktypes.UpdateGroupMemberOption{TxOpts: txOpts})
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txnHash, "UpdateGroupMember")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("update_group: %s \ntransaction hash: %s\n", groupName, txnHash)
	return nil
}

func renewGroupMember(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	renewMembersInfo := ctx.String(renewMemberFlag)

	if renewMembersInfo == "" {
		return toCmdErr(errors.New("fail to get members to renew"))
	}

	var renewGroupMembers []string
	if strings.Contains(renewMembersInfo, ",") {
		renewGroupMembers = strings.Split(renewMembersInfo, ",")
	} else if renewMembersInfo != "" {
		renewGroupMembers = []string{renewMembersInfo}
	}

	groupOwner, err := getGroupOwner(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	expireTimestamp := ctx.Int64(expireTimeFlag)
	if expireTimestamp < time.Now().Unix() {
		return toCmdErr(errors.New("expire stamp should be more than" + strconv.Itoa(int(time.Now().Unix()))))
	}

	memberNum := len(renewGroupMembers)
	expireTimeList := make([]time.Time, memberNum)
	for i := 0; i < memberNum; i++ {
		expireTimeList[i] = time.Unix(expireTimestamp, 0)
	}

	c, cancelUpdateGroup := context.WithCancel(globalContext)
	defer cancelUpdateGroup()

	_, err = client.HeadGroup(c, groupName, groupOwner)
	if err != nil {
		return toCmdErr(ErrGroupNotExist)
	}

	txOpts := &types.TxOption{Mode: &SyncBroadcastMode}
	txnHash, err := client.RenewGroupMember(c, groupOwner, groupName, renewGroupMembers, expireTimeList,
		sdktypes.RenewGroupMemberOption{TxOpts: txOpts})
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txnHash, "renewGroupMember")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("renew_group: %s \ntransaction hash: %s\n", groupName, txnHash)
	return nil
}

func getGroupOwner(ctx *cli.Context) (string, error) {
	groupOwnerAddrStr := ctx.String(groupOwnerFlag)

	if groupOwnerAddrStr != "" {
		return groupOwnerAddrStr, nil
	}

	keyJson, _, err := loadKeyStoreFile(ctx)
	if err != nil {
		return "", err
	}

	k := new(encryptedKey)
	if err = json.Unmarshal(keyJson, k); err != nil {
		return "", err
	}

	return k.Address, nil
}

func mirrorGroup(ctx *cli.Context) error {
	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}
	id := math.NewUint(0)
	if ctx.String(IdFlag) != "" {
		id = math.NewUintFromString(ctx.String(IdFlag))
	}

	groupName := ctx.String(groupNameFlag)

	c, cancelContext := context.WithCancel(globalContext)
	defer cancelContext()

	txResp, err := client.MirrorGroup(c, id, groupName, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("mirror_group: %s \ntransaction hash: %s\n", groupName, txResp.TxHash)
	return nil
}
