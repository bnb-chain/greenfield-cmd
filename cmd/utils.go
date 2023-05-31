package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/BurntSushi/toml"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/eth/ethsecp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

const (
	maxFileSize      = 2 * 1024 * 1024 * 1024
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
	groupIDFlag      = "groupId"
	granteeFlag      = "grantee"
	actionsFlag      = "actions"
	effectFlag       = "effect"
	expireTimeFlag   = "expire"

	ownerAddressFlag = "owner"
	addressFlag      = "address"
	toAddressFlag    = "toAddress"
	fromAddressFlag  = "fromAddress"
	amountFlag       = "amount"
	resourceFlag     = "resource"
	IdFlag           = "id"
	objectPrefix     = "prefix"
	folderFlag       = "folder"

	groupNameFlag  = "groupName"
	bucketNameFlag = "bucketName"
	objectNameFlag = "objectName"

	defaultKeyfile      = "key.json"
	defaultPasswordfile = "password"
	privKeyFileFlag     = "privKeyFile"
	passwordFileFlag    = "passwordfile"
	EncryptScryptN      = 1 << 18
	EncryptScryptP      = 1

	ContextTimeout = time.Second * 20
)

var (
	ErrBucketNotExist     = errors.New("bucket not exist")
	ErrObjectNotExist     = errors.New("object not exist")
	ErrObjectNotCreated   = errors.New("object not created on chain")
	ErrObjectSeal         = errors.New("object not sealed before downloading")
	ErrGroupNotExist      = errors.New("group not exist")
	ErrFileNotExist       = errors.New("file path not exist")
	SyncBroadcastMode     = tx.BroadcastMode_BROADCAST_MODE_SYNC
	TxnOptionWithSyncMode = types.TxOption{Mode: &SyncBroadcastMode}
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
		return storageTypes.VISIBILITY_TYPE_UNSPECIFIED, errors.New("invalid visibility type")
	}
}

func toCmdErr(err error) error {
	fmt.Printf("run command error: %s\n", err.Error())
	return nil
}

func genCmdErr(msg string) error {
	fmt.Printf("run command error: %s\n", msg)
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

func parsePrincipal(grantee string, groupId uint64) (sdktypes.Principal, error) {
	if grantee == "" && groupId == 0 {
		return "", errors.New("group id or account need to be set")
	}

	if grantee != "" && groupId > 0 {
		return "", errors.New("not support setting group id and account at the same time")
	}

	var principal sdktypes.Principal
	var granteeAddr sdk.AccAddress
	var err error
	if groupId > 0 {
		p := permTypes.NewPrincipalWithGroup(sdkmath.NewUint(groupId))
		principalBytes, err := p.Marshal()
		if err != nil {
			return "", err
		}
		principal = sdktypes.Principal(principalBytes)
	} else {
		granteeAddr, err = sdk.AccAddressFromHexUnsafe(grantee)
		if err != nil {
			return "", err
		}
		p := permTypes.NewPrincipalWithAccount(granteeAddr)
		principalBytes, err := p.Marshal()
		if err != nil {
			return "", err
		}
		principal = sdktypes.Principal(principalBytes)
	}

	return principal, nil
}

func getBucketAction(action string) (permTypes.ActionType, error) {
	switch action {
	case "update":
		return permTypes.ACTION_UPDATE_BUCKET_INFO, nil
	case "delete":
		return permTypes.ACTION_DELETE_BUCKET, nil
	case "create":
		return permTypes.ACTION_CREATE_OBJECT, nil
	case "list":
		return permTypes.ACTION_LIST_OBJECT, nil
	case "createObj":
		return permTypes.ACTION_CREATE_OBJECT, nil
	case "deleteObj":
		return permTypes.ACTION_DELETE_OBJECT, nil
	case "copyObj":
		return permTypes.ACTION_COPY_OBJECT, nil
	case "getObj":
		return permTypes.ACTION_GET_OBJECT, nil
	case "executeObj":
		return permTypes.ACTION_EXECUTE_OBJECT, nil
	case "all":
		return permTypes.ACTION_TYPE_ALL, nil
	default:
		return permTypes.ACTION_UNSPECIFIED, errors.New("invalid action :" + action)
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
	case "list":
		return permTypes.ACTION_LIST_OBJECT, nil
	case "all":
		return permTypes.ACTION_TYPE_ALL, nil
	default:
		return permTypes.ACTION_UNSPECIFIED, errors.New("invalid action:" + action)
	}
}

func parseActions(ctx *cli.Context, isObjectPolicy bool) ([]permTypes.ActionType, error) {
	actions := make([]permTypes.ActionType, 0)
	actionListStr := ctx.String(actionsFlag)
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

// getPassword return the password content
func getPassword(ctx *cli.Context, config *cmdConfig) (string, error) {
	var filepath string
	if passwordFile := ctx.String(passwordFileFlag); passwordFile != "" {
		filepath = passwordFile
	} else if config.PasswordFile != "" {
		filepath = config.PasswordFile
	} else {
		filepath = defaultPasswordfile
	}

	readContent, err := os.ReadFile(filepath)
	if err != nil {
		return "", errors.New("failed to read password file" + err.Error())
	}

	return strings.TrimRight(string(readContent), "\r\n"), nil
}

// loadKey loads a secp256k1 private key from the given file.
func loadKey(file string) (string, sdk.AccAddress, error) {
	fd, err := os.Open(file)
	if err != nil {
		return "", nil, err
	}

	r := bufio.NewReader(fd)
	buf := make([]byte, 64)
	var n int
	for ; n < len(buf); n++ {
		buf[n], err = r.ReadByte()
		switch {
		case err == io.EOF || buf[n] < '!':
			break
		case err != nil:
			return "", nil, err
		}
	}
	if n != len(buf) {
		return "", nil, fmt.Errorf("key file too short, want 42 hex characters")
	}

	priBytes, err := hex.DecodeString(string(buf))
	if err != nil {
		return "", nil, err
	}

	if len(priBytes) != 32 {
		return "", nil, fmt.Errorf("Len of Keybytes is not equal to 32 ")
	}
	var keyBytesArray [32]byte
	copy(keyBytesArray[:], priBytes[:32])
	priKey := hd.EthSecp256k1.Generate()(keyBytesArray[:]).(*ethsecp256k1.PrivKey)

	return string(buf), sdk.AccAddress(priKey.PubKey().Address()), nil
}

type cmdConfig struct {
	RpcAddr      string `toml:"rpcAddr"`
	ChainId      string `toml:"chainId"`
	PasswordFile string `toml:"passwordFile"`
	Host         string `toml:"host"`
}

func parseConfigFile(filePath string) (*cmdConfig, error) {
	var config cmdConfig
	if _, err := toml.DecodeFile(filePath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
