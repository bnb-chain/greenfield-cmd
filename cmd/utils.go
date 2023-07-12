package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	sdkutils "github.com/bnb-chain/greenfield-go-sdk/pkg/utils"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/eth/ethsecp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

const (
	Version          = "v0.0.8-hf.1"
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
	IdFlag           = "id"

	ownerAddressFlag = "owner"
	addressFlag      = "address"
	toAddressFlag    = "toAddress"
	fromAddressFlag  = "fromAddress"
	amountFlag       = "amount"
	objectPrefix     = "prefix"
	folderFlag       = "folder"

	privKeyFileFlag  = "privKeyFile"
	privKeyFlag      = "privateKey"
	passwordFileFlag = "passwordfile"
	homeFlag         = "home"
	keyStoreFlag     = "keystore"
	configFlag       = "config"
	EncryptScryptN   = 1 << 18
	EncryptScryptP   = 1

	ContextTimeout       = time.Second * 20
	BucketResourcePrefix = "grn:b::"
	ObjectResourcePrefix = "grn:o::"
	GroupResourcePrefix  = "grn:g:"

	ObjectResourceType = 1
	BucketResourceType = 2
	GroupResourceType  = 3

	DefaultConfigPath   = "config/config.toml"
	DefaultConfigDir    = ".gnfd-cmd"
	DefaultKeyStorePath = "keystore/key.json"
	DefaultPasswordPath = "keystore/password/password.txt"

	rpcAddrConfigField = "rpcAddr"
	chainIdConfigField = "chainId"
	hostConfigField    = "host"
	groupNameFlag      = "groupName"
	bucketNameFlag     = "bucketName"
	objectNameFlag     = "objectName"
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

	return ctx.Args().Get(0), nil
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
		principal, err = sdkutils.NewPrincipalWithGroupId(groupId)
		if err != nil {
			return "", err
		}
	} else {
		granteeAddr, err = sdk.AccAddressFromHexUnsafe(grantee)
		if err != nil {
			return "", err
		}
		principal, err = sdkutils.NewPrincipalWithAccount(granteeAddr)
		if err != nil {
			return "", err
		}
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
	case "update":
		return permTypes.ACTION_UPDATE_OBJECT_INFO, nil
	case "all":
		return permTypes.ACTION_TYPE_ALL, nil
	default:
		return permTypes.ACTION_UNSPECIFIED, errors.New("invalid action:" + action)
	}
}

func getGroupAction(action string) (permTypes.ActionType, error) {
	switch action {
	case "update":
		return permTypes.ACTION_UPDATE_GROUP_MEMBER, nil
	case "delete":
		return permTypes.ACTION_DELETE_GROUP, nil
	case "all":
		return permTypes.ACTION_TYPE_ALL, nil
	default:
		return permTypes.ACTION_UNSPECIFIED, errors.New("invalid action:" + action)
	}
}

func parseActions(ctx *cli.Context, resourceType ResourceType) ([]permTypes.ActionType, error) {
	actions := make([]permTypes.ActionType, 0)
	actionListStr := ctx.String(actionsFlag)
	if actionListStr == "" {
		return nil, errors.New("fail to parse actions")
	}

	actionList := strings.Split(actionListStr, ",")
	for _, v := range actionList {
		var action permTypes.ActionType
		var err error
		if resourceType == ObjectResourceType {
			action, err = getObjectAction(v)
		} else if resourceType == BucketResourceType {
			action, err = getBucketAction(v)
		} else if resourceType == GroupResourceType {
			action, err = getGroupAction(v)
		}

		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// getPassword return the password content
func getPassword(ctx *cli.Context) (string, error) {
	var filepath string
	if passwordFile := ctx.String(passwordFileFlag); passwordFile != "" {
		filepath = passwordFile
		readContent, err := os.ReadFile(filepath)
		if err != nil {
			return "", errors.New("failed to read password file" + err.Error())
		}
		return strings.TrimRight(string(readContent), "\r\n"), nil
	}

	fmt.Print("Input Password：")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("read password ", err)
		return "", err
	}
	password := string(bytePassword)

	return password, nil
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
	RpcAddr string `toml:"rpcAddr"`
	ChainId string `toml:"chainId"`
	Host    string `toml:"host"`
}

// parseConfigFile decode the config file of TOML format
func parseConfigFile(filePath string) (*cmdConfig, error) {
	var config cmdConfig
	if _, err := toml.DecodeFile(filePath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

const configContent = "rpcAddr = \"https://gnfd-testnet-fullnode-tendermint-us.bnbchain.org:443\"\nchainId = \"greenfield_5600-1\""

// loadConfig parse the default config file path
func loadConfig(ctx *cli.Context) (*cmdConfig, error) {
	homeDir, err := getHomeDir(ctx)
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(homeDir, DefaultConfigPath)

	// if config default path not exist, create the config file with default test net config
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
			return nil, toCmdErr(errors.New("failed to create config file directory :%s" + filepath.Dir(configPath)))
		}

		err = os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create config file: %v", err)
		}
		fmt.Println("generate default config file on path:", configPath)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check config file: %v", err)
	}

	content, err := parseConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	return content, nil
}

// getConfig parse the config of the client, return rpc address, chainId and host
func getConfig(ctx *cli.Context) (string, string, string, error) {
	rpcAddr := ctx.String(rpcAddrConfigField)
	chainId := ctx.String(chainIdConfigField)
	if rpcAddr != "" && chainId != "" {
		return rpcAddr, chainId, ctx.String(hostConfigField), nil
	}

	configFile := ctx.String("config")
	var config *cmdConfig
	var err error
	if configFile != "" {
		// if user has set config file, parse the file
		config, err = parseConfigFile(configFile)
		if err != nil {
			return "", "", "", err
		}
	} else {
		// if file exist in config default path, read default file.
		// else generate the default file for user in the default path
		config, err = loadConfig(ctx)
		if err != nil {
			return "", "", "", err
		}
	}

	if config.RpcAddr == "" || config.ChainId == "" {
		return "", "", "", fmt.Errorf("failed to parse rpc address or chain id , please set it in the config file")
	}

	return config.RpcAddr, config.ChainId, config.Host, nil
}

func loadKeyStoreFile(ctx *cli.Context) ([]byte, error) {
	keyfilepath := ctx.String("keystore")
	if keyfilepath == "" {
		homeDir, err := getHomeDir(ctx)
		if err != nil {
			return nil, err
		}
		keyfilepath = filepath.Join(homeDir, DefaultKeyStorePath)
	}

	// fetch private key from keystore
	content, err := os.ReadFile(keyfilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the keyfile at '%s': %v \n", keyfilepath, err)
	}

	return content, nil
}

func loadPassWordFile(ctx *cli.Context) (string, error) {
	passwordFilepath := ctx.String(passwordFileFlag)
	if passwordFilepath == "" {
		homeDir, err := getHomeDir(ctx)
		if err != nil {
			return "", err
		}
		passwordFilepath = filepath.Join(homeDir, DefaultPasswordPath)
	}

	// fetch password from password file
	content, err := os.ReadFile(passwordFilepath)
	if err != nil {
		return "", fmt.Errorf("failed to read the password at '%s': %v \n", passwordFilepath, err)
	}

	return string(content), nil
}

func getHomeDir(ctx *cli.Context) (string, error) {
	if ctx.String(homeFlag) != "" {
		return ctx.String(homeFlag), nil
	}
	return "", errors.New("home flag should not be empty")
}
