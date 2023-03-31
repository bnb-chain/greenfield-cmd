package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

const (
	maxFileSize          = 5 * 1024 * 1024 * 1024
	publicReadType       = "public-read"
	privateType          = "private"
	inheritType          = "inherit"
	primarySPFlagName    = "primarySP"
	chargeQuotaFlagName  = "chargedQuota"
	visibilityFlagName   = "visibility"
	paymentFlagName      = "paymentAddress"
	secondarySPFlagName  = "secondarySPs"
	contentTypeFlagName  = "contentType"
	txnHashFlagName      = "txnHash"
	startOffsetFlagName  = "start"
	endOffsetFlagName    = "end"
	initMemberFlagName   = "initMembers"
	addMemberFlagName    = "addMembers"
	removeMemberFlagName = "removeMembers"
	groupOwnerFlagName   = "groupOwner"
	spAddressFlagName    = "spAddress"
	objectIDFlagName     = "objectId"
	pieceIndexFlagName   = "pieceIndex"
	spIndexFlagName      = "spIndex"
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
