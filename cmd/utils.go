package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/eth/ethsecp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/cosmos/gogoproto/proto"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	sdkutils "github.com/bnb-chain/greenfield-go-sdk/pkg/utils"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
)

const (
	Version                 = "v1.0.2"
	maxFileSize             = 64 * 1024 * 1024 * 1024
	maxPutWithoutResumeSize = 2 * 1024 * 1024 * 1024
	publicReadType          = "public-read"
	privateType             = "private"
	inheritType             = "inherit"
	effectAllow             = "allow"
	effectDeny              = "deny"
	primarySPFlag           = "primarySP"
	chargeQuotaFlag         = "chargedQuota"
	visibilityFlag          = "visibility"
	tagFlag                 = "tags"
	paymentFlag             = "paymentAddress"
	secondarySPFlag         = "secondarySPs"
	contentTypeFlag         = "contentType"
	startOffsetFlag         = "start"
	endOffsetFlag           = "end"
	recursiveFlag           = "recursive"
	bypassSealFlag          = "bypassSeal"
	addMemberFlag           = "addMembers"
	removeMemberFlag        = "removeMembers"
	renewMemberFlag         = "renewMembers"
	groupOwnerFlag          = "groupOwner"
	groupMemberExpireFlag   = "expireTime"
	groupIDFlag             = "groupId"
	granteeFlag             = "grantee"
	actionsFlag             = "actions"
	effectFlag              = "effect"
	expireTimeFlag          = "expire"
	IdFlag                  = "id"
	DestChainIdFlag         = "destChainId"

	ownerAddressFlag = "owner"
	addressFlag      = "address"
	toAddressFlag    = "toAddress"
	fromAddressFlag  = "fromAddress"
	amountFlag       = "amount"

	unsafeFlag       = "unsafe"
	unarmoredFlag    = "unarmoredHex"
	passwordFileFlag = "passwordfile"
	formatFlag       = "format"
	defaultFormat    = "plaintxt"
	jsonFormat       = "json"
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

	DefaultConfigPath  = "config/config.toml"
	DefaultConfigDir   = ".gnfd-cmd"
	DefaultAccountPath = "account/defaultKey"
	DefaultKeyDir      = "keystore"

	rpcAddrConfigField = "rpcAddr"
	chainIdConfigField = "chainId"
	hostConfigField    = "host"
	groupNameFlag      = "groupName"
	bucketNameFlag     = "bucketName"
	objectNameFlag     = "objectName"

	// resumable download & upload
	partSizeFlag  = "partSize"
	resumableFlag = "resumable"

	// fast download
	fastFlag = "fast"

	// download with specified sp host
	spHostFlag = "spHost"

	operatorAddressLen = 42
	accountAddressLen  = 40
	exitStatus         = "GRACEFUL_EXITING"
	StatusSPrefix      = "STATUS_"
	defaultMaxKey      = 500

	noBalanceErr           = "key not found"
	maxListMemberNum       = 1000
	progressDelayPrintSize = 10 * 1024 * 1024
	timeFormat             = "2006-01-02T15-04-05.000000000Z"

	printRateInterval  = time.Second / 2
	bytesToReadForMIME = 512
	notFound           = -1
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

// ClientOptions indicates the metadata to construct new greenfield client
type ClientOptions struct {
	IsQueryCmd bool   // indicate whether the command is query command
	Endpoint   string // indicates the endpoint of sp
}

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
	if strings.Contains(err.Error(), noBalanceErr) {
		fmt.Println("The operator account have no balance, please transfer token to your account")
	} else {
		fmt.Printf("run command error: %s\n", err.Error())
	}
	return nil
}

// parse object info meta on the chain
func parseObjectInfo(objectDetail *sdktypes.ObjectDetail) {
	info := objectDetail.ObjectInfo.String()
	fmt.Println("object_status:", objectDetail.ObjectInfo.ObjectStatus)
	infoStr := strings.Split(info, " ")
	checksumID := 0
	for _, objInfo := range infoStr {
		if strings.Contains(objInfo, "create_at:") {
			timeInfo := strings.Split(objInfo, ":")
			timestamp, _ := strconv.ParseInt(timeInfo[1], 10, 64)
			location, _ := time.LoadLocation("Asia/Shanghai")
			t := time.Unix(timestamp, 0).In(location)
			objInfo = timeInfo[0] + ":" + t.Format(iso8601DateFormat)
		}
		if strings.Contains(objInfo, "checksums:") {
			if checksumID == 0 {
				fmt.Println("checksums:")
			}
			hashInfo := strings.Split(objInfo, ":")
			objInfo = hashInfo[0] + "[" + strconv.Itoa(checksumID) + "]" + ":" + hex.EncodeToString([]byte(hashInfo[1]))
			checksumID++
		}
		if strings.Contains(objInfo, "status") {
			continue
		}
		fmt.Println(objInfo)
	}
}

func getJsonMarshaler() *jsonpb.Marshaler {
	// Create a JSON marshaler
	return &jsonpb.Marshaler{
		EmitDefaults: true, // Include fields with zero values
		OrigName:     true,
	}
}

func parseByJsonFormat(v proto.Message) {
	jsonData, err := getJsonMarshaler().MarshalToString(v)
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}
	fmt.Println(jsonData)
}

func parseBucketInfo(info *storageTypes.BucketInfo) {
	fmt.Println("bucket_status:", info.BucketStatus.String())
	infoStr := strings.Split(info.String(), " ")
	for _, bucketInfo := range infoStr {
		if strings.Contains(bucketInfo, "create_at:") {
			timeInfo := strings.Split(bucketInfo, ":")
			timestamp, _ := strconv.ParseInt(timeInfo[1], 10, 64)
			location, _ := time.LoadLocation("Asia/Shanghai")
			t := time.Unix(timestamp, 0).In(location)
			bucketInfo = timeInfo[0] + ":" + t.Format(iso8601DateFormat)
		}
		fmt.Println(bucketInfo)
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
func getBucketAction(action string) (permTypes.ActionType, bool, error) {
	switch action {
	case "update":
		return permTypes.ACTION_UPDATE_BUCKET_INFO, false, nil
	case "delete":
		return permTypes.ACTION_DELETE_BUCKET, false, nil
	case "list":
		return permTypes.ACTION_LIST_OBJECT, false, nil
	case "createObj":
		return permTypes.ACTION_CREATE_OBJECT, true, nil
	case "deleteObj":
		return permTypes.ACTION_DELETE_OBJECT, true, nil
	case "copyObj":
		return permTypes.ACTION_COPY_OBJECT, true, nil
	case "getObj":
		return permTypes.ACTION_GET_OBJECT, true, nil
	case "executeObj":
		return permTypes.ACTION_EXECUTE_OBJECT, true, nil
	case "all":
		return permTypes.ACTION_TYPE_ALL, true, nil
	default:
		return permTypes.ACTION_UNSPECIFIED, false, errors.New("invalid action :" + action)
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

func parseActions(ctx *cli.Context, resourceType ResourceType) ([]permTypes.ActionType, bool, error) {
	actions := make([]permTypes.ActionType, 0)
	actionListStr := ctx.String(actionsFlag)
	if actionListStr == "" {
		return nil, false, errors.New("fail to parse actions")
	}

	actionList := strings.Split(actionListStr, ",")
	var isObjectActionInBucketPolicy bool
	for _, v := range actionList {
		var action permTypes.ActionType
		var err error
		if resourceType == ObjectResourceType {
			action, err = getObjectAction(v)
		} else if resourceType == BucketResourceType {
			action, isObjectActionInBucketPolicy, err = getBucketAction(v)
		} else if resourceType == GroupResourceType {
			action, err = getGroupAction(v)
		}

		if err != nil {
			return nil, isObjectActionInBucketPolicy, err
		}
		actions = append(actions, action)
	}

	return actions, isObjectActionInBucketPolicy, nil
}

// getPassword return the password content
func getPassword(ctx *cli.Context, needNotice bool) (string, error) {
	var filepath string
	if passwordFile := ctx.String(passwordFileFlag); passwordFile != "" {
		filepath = passwordFile
		readContent, err := os.ReadFile(filepath)
		if err != nil {
			return "", errors.New("failed to read password file" + err.Error())
		}
		return strings.TrimRight(string(readContent), "\r\n"), nil
	}

	fmt.Print("Please enter the passphrase now:")

	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("read password err:", err)
		return "", err
	}
	password := string(bytePassword)
	fmt.Println()
	if needNotice {
		fmt.Println("- You must BACKUP your key file! Without the key, it's impossible to set transaction to greenfield!")
		fmt.Println("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!")
	}
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

const configContent = "rpcAddr = \"https://greenfield-chain.bnbchain.org:443\"\nchainId = \"greenfield_1017-1\""

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

func loadKeyStoreFile(ctx *cli.Context) ([]byte, string, error) {
	keyfilePath := ctx.String("keystore")
	if keyfilePath == "" {
		homeDir, err := getHomeDir(ctx)
		if err != nil {
			return nil, "", err
		}

		defaultAddrFilePath := filepath.Join(homeDir, DefaultAccountPath)
		fileContent, err := os.ReadFile(defaultAddrFilePath)
		if err != nil {
			return nil, "", fmt.Errorf("invalid default address" + err.Error())
		}
		if len(fileContent) != accountAddressLen {
			return nil, "", fmt.Errorf("invalid default address length")
		}
		// get the default keystore file path
		keyStorePath := filepath.Join(homeDir, DefaultKeyDir)
		keyfilePath, err = getKeystoreFileByAddress(keyStorePath, string(fileContent))
		if err != nil {
			return nil, "", fmt.Errorf("failed to load the default keystore:" + err.Error())
		}
	}

	// fetch private key from keystore
	content, err := os.ReadFile(keyfilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the keyfile at '%s': %v \n", keyfilePath, err)
	}

	return content, keyfilePath, nil
}

// getKeystoreFileByAddress list the keystore dir and find the file with address suffix
func getKeystoreFileByAddress(directory string, address string) (string, error) {
	var filePath string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), address) {
			filePath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return filePath, nil
}

func getHomeDir(ctx *cli.Context) (string, error) {
	if ctx.String(homeFlag) != "" {
		return ctx.String(homeFlag), nil
	}
	return "", errors.New("home flag should not be empty")
}

func getUserAddress(ctx *cli.Context) (string, error) {
	var userAddress string
	var err error
	flagAddr := ctx.String(addressFlag)
	if flagAddr != "" {
		_, err = sdk.AccAddressFromHexUnsafe(flagAddr)
		if err != nil {
			return "", toCmdErr(err)
		}
		userAddress = flagAddr
	} else {
		keyJson, _, err := loadKeyStoreFile(ctx)
		if err != nil {
			return "", toCmdErr(err)
		}

		k := new(encryptedKey)
		if err = json.Unmarshal(keyJson, k); err != nil {
			return "", toCmdErr(errors.New("failed to get account info: " + err.Error()))
		}
		userAddress = k.Address
	}
	return userAddress, nil
}

func parseFileByArg(ctx *cli.Context, argIndex int) (int64, error) {
	exists, objectSize, err := pathExists(ctx.Args().Get(argIndex))
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, fmt.Errorf("upload file not exists")
	} else if objectSize > int64(maxFileSize) {
		return 0, fmt.Errorf("upload file larger than 10G ")
	}
	return objectSize, nil
}

type ProgressReader struct {
	io.Reader
	Total          int64
	Current        int64
	StartTime      time.Time
	LastPrinted    time.Time
	LastPrintedStr string
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	pr.printProgress()
	return n, err
}

func (pr *ProgressReader) printProgress() {
	progress := float64(pr.Current) / float64(pr.Total) * 100
	now := time.Now()
	elapsed := now.Sub(pr.StartTime)
	uploadSpeed := float64(pr.Current) / elapsed.Seconds()

	if now.Sub(pr.LastPrinted) >= printRateInterval { // print rate every half second
		progressStr := fmt.Sprintf("uploading progress: %.2f%% [ %s / %s ], rate: %s    ",
			progress, getConvertSize(pr.Current), getConvertSize(pr.Total), getConvertRate(uploadSpeed))
		// Clear current line
		fmt.Print("\r", strings.Repeat(" ", len(pr.LastPrintedStr)), "\r")
		// Print new progress
		fmt.Print(progressStr)

		pr.LastPrinted = now
	}
}

type ProgressWriter struct {
	io.Writer
	Total       int64
	Current     int64
	StartTime   time.Time
	LastPrinted time.Time
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Current += int64(n)
	pw.printProgress()
	return n, err
}

func (pw *ProgressWriter) printProgress() {
	progress := float64(pw.Current) / float64(pw.Total) * 100
	now := time.Now()

	elapsed := now.Sub(pw.StartTime)
	downloadedBytes := pw.Current
	downloadSpeed := float64(downloadedBytes) / elapsed.Seconds()

	if now.Sub(pw.LastPrinted) >= printRateInterval { // print rate every half second
		fmt.Printf("\rdownloding progress: %.2f%% [ %s / %s ], rate: %s    ",
			progress, getConvertSize(pw.Current), getConvertSize(pw.Total), getConvertRate(downloadSpeed))
		pw.LastPrinted = now
	}
}

func getConvertSize(fileSize int64) string {
	var convertedSize string
	if fileSize > 1<<30 {
		convertedSize = fmt.Sprintf("%.2fG", float64(fileSize)/(1<<30))
	} else if fileSize > 1<<20 {
		convertedSize = fmt.Sprintf("%.2fM", float64(fileSize)/(1<<20))
	} else if fileSize > 1<<10 {
		convertedSize = fmt.Sprintf("%.2fK", float64(fileSize)/(1<<10))
	} else {
		convertedSize = fmt.Sprintf("%dB", fileSize)
	}
	return convertedSize
}

func getConvertRate(rate float64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case rate >= MB:
		return fmt.Sprintf("%.2f MB/s", rate/MB)
	case rate >= KB:
		return fmt.Sprintf("%.2f KB/s", rate/KB)
	default:
		return fmt.Sprintf("%.2f Byte/s", rate)
	}
}

func checkIfDownloadFileExist(filePath, objectName string) (string, error) {
	st, err := os.Stat(filePath)
	if err == nil {
		// If the destination exists and is a directory.
		if st.IsDir() {
			filePath = filePath + "/" + objectName
			return filePath, nil
		}
		return filePath, fmt.Errorf("download file:%s already exist\n", filePath)
	}
	return filePath, nil
}

func convertAddressToLower(str string) string {
	converted := strings.Map(func(r rune) rune {
		if unicode.IsUpper(r) {
			return unicode.ToLower(r)
		}
		return r
	}, str)

	if strings.HasPrefix(str, "0x") {
		converted = converted[2:]
	}
	return converted
}

func getContentTypeOfFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// read the first bits of file for judgment of the mime type
	buffer := make([]byte, bytesToReadForMIME)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	return contentType, nil
}
