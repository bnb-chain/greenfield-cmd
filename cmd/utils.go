package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

const (
	maxFileSize      = 5 * 1024 * 1024 * 1024
	maxListObjects   = 100
	publicReadType   = "public-read"
	privateType      = "private"
	inheritType      = "inherit"
	effectAllow      = "allow"
	effectDeny       = "deny"
	primarySPFlag    = "primarySP"
	chargeQuotaFlag  = "chargedQuota"
	visibilityFlag   = "visibility"
	paymentFlag      = "paymentAddress"
	secondarySPFlag  = "secondarySPs"
	contentTypeFlag  = "contentType"
	startOffsetFlag  = "start"
	endOffsetFlag    = "end"
	initMemberFlag   = "initMembers"
	addMemberFlag    = "addMembers"
	removeMemberFlag = "removeMembers"
	groupOwnerFlag   = "groupOwner"
	headMemberFlag   = "headMember"
	spAddressFlag    = "spAddress"
	groupIDFlag      = "groupId"
	granterFlag      = "granter"
	actionsFlag      = "actions"
	effectFlag       = "effect"
	userAddressFlag  = "user"
	expireTimeFlag   = "expire"

	ownerAddressFlag = "owner"
	addressFlag      = "address"
	toAddressFlag    = "toAddress"
	fromAddressFlag  = "fromAddress"
	amountFlag       = "amount"
	resourceFlag     = "resource"
	IdFlag           = "id"
)

var (
	ErrBucketNotExist   = errors.New("bucket not exist")
	ErrObjectNotExist   = errors.New("object not exist")
	ErrObjectNotCreated = errors.New("object not created on chain")
	ErrObjectSeal       = errors.New("object not sealed before downloading")
	ErrGroupNotExist    = errors.New("group not exist")
)

type CmdEnumValue struct {
	Enum     []string
	Default  string
	selected string
}

func (e *CmdEnumValue) Set(value string) error {
	for _, enum := range e.Enum {
		if enum == value {
			e.selected = value
			return nil
		}
	}

	return fmt.Errorf("allowed values are %s", strings.Join(e.Enum, ", "))
}

func (e CmdEnumValue) String() string {
	if e.selected == "" {
		return e.Default
	}
	return e.selected
}

func getVisibilityType(visibility string) (storageTypes.VisibilityType, error) {
	switch visibility {
	case publicReadType:
		return storageTypes.VISIBILITY_TYPE_PUBLIC_READ, nil
	case privateType:
		return storageTypes.VISIBILITY_TYPE_PRIVATE, nil
	case inheritType:
		return storageTypes.VISIBILITY_TYPE_INHERIT, nil
	default:
		return storageTypes.VISIBILITY_TYPE_PRIVATE, errors.New("invalid visibility type")
	}
}

func toCmdErr(err error) error {
	fmt.Printf("run command error: %s\n", err.Error())
	return nil
}

// parse bucket info or object info meta on the chain
func parseChainInfo(info string, isBucketInfo bool) {
	if isBucketInfo {
		fmt.Println("latest bucket info:")
	} else {
		fmt.Println("latest object info:")
	}
	infoStr := strings.Split(info, " ")
	for _, info := range infoStr {
		if strings.Contains(info, "create_at:") {
			timeInfo := strings.Split(info, ":")
			timestamp, _ := strconv.ParseInt(timeInfo[1], 10, 64)
			location, _ := time.LoadLocation("Asia/Shanghai")
			t := time.Unix(timestamp, 0).In(location)
			info = timeInfo[0] + ":" + t.Format(iso8601DateFormatSecond)
		}
		if strings.Contains(info, "checksums:") {
			hashInfo := strings.Split(info, ":")
			info = hashInfo[0] + ":" + hex.EncodeToString([]byte(hashInfo[1]))
		}
		fmt.Println(info)
	}
}

func getBucketNameByUrl(ctx *cli.Context) (string, error) {
	if ctx.NArg() < 1 {
		return "", errors.New("the args should be more than one")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName := ParseBucket(urlInfo)

	if bucketName == "" {
		return "", errors.New("fail to parse bucket name")
	}
	return bucketName, nil
}

func getGroupNameByUrl(ctx *cli.Context) (string, error) {
	if ctx.NArg() < 1 {
		return "", errors.New("the args should be more than one")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName := ParseBucket(urlInfo)

	if bucketName == "" {
		return "", errors.New("fail to parse group name")
	}
	return bucketName, nil
}

func parseAddrList(addrInfo string) ([]sdk.AccAddress, error) {
	addresses := strings.Split(addrInfo, ",")
	addrList := make([]sdk.AccAddress, len(addresses))
	var err error
	for idx, addr := range addresses {
		addrList[idx], err = sdk.AccAddressFromHexUnsafe(addr)
		if err != nil {
			return nil, toCmdErr(err)
		}
	}
	return addrList, nil
}

func parsePrincipal(ctx *cli.Context, granter string, groupId uint64) (gnfdclient.Principal, error) {
	if granter == "" && groupId == 0 {
		return "", errors.New("group id or account need to be set")
	}

	if granter != "" && groupId > 0 {
		return "", errors.New("not support setting group id and account at the same time")
	}

	var principal gnfdclient.Principal
	var granterAddr sdk.AccAddress
	var err error
	if groupId > 0 {
		p := permTypes.NewPrincipalWithGroup(sdkmath.NewUint(groupId))
		principalBytes, err := p.Marshal()
		if err != nil {
			return "", err
		}
		principal = gnfdclient.Principal(principalBytes)
	} else {
		granterAddr, err = sdk.AccAddressFromHexUnsafe(granter)
		if err != nil {
			return "", err
		}
		p := permTypes.NewPrincipalWithAccount(granterAddr)
		principalBytes, err := p.Marshal()
		if err != nil {
			return "", err
		}
		principal = gnfdclient.Principal(principalBytes)
	}

	return principal, nil
}

func getBucketAction(action string) (permTypes.ActionType, error) {
	switch action {
	case "update":
		return permTypes.ACTION_UPDATE_BUCKET_INFO, nil
	case "delete":
		return permTypes.ACTION_DELETE_BUCKET, nil
	default:
		return permTypes.ACTION_EXECUTE_OBJECT, errors.New("invalid action of bucket policy")
	}
}

func getObjectAction(action string) (permTypes.ActionType, error) {
	switch action {
	case "create":
		return permTypes.ACTION_CREATE_OBJECT, nil
	case "delete":
		return permTypes.ACTION_DELETE_OBJECT, nil
	case "copy":
		return permTypes.ACTION_COPY_OBJECT, nil
	case "get":
		return permTypes.ACTION_GET_OBJECT, nil
	case "execute":
		return permTypes.ACTION_EXECUTE_OBJECT, nil
	default:
		return permTypes.ACTION_EXECUTE_OBJECT, errors.New("invalid action of object policy")
	}
}

func parseActions(ctx *cli.Context, isObjectPolicy bool) ([]permTypes.ActionType, error) {
	actions := make([]permTypes.ActionType, 0)
	actionListStr := ctx.String(actionsFlagName)
	if actionListStr == "" {
		return nil, errors.New("fail to parse actions")
	}

	actionList := strings.Split(actionListStr, ",")
	for _, v := range actionList {
		var action permTypes.ActionType
		var err error
		if isObjectPolicy {
			action, err = getObjectAction(v)
		} else {
			action, err = getBucketAction(v)
		}

		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}
