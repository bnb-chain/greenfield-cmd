package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	spClient "github.com/bnb-chain/greenfield-go-sdk/client/sp"
	"github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/urfave/cli/v2"
)

// cmdCreateObj send the request get approval of uploading
func cmdCreateObj() *cli.Command {
	return &cli.Command{
		Name:      "create-object",
		Action:    createObject,
		Usage:     "create an object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Get approval from storage provider and send createObject txn to chain.
The command need to pass the file path inorder to compute hash roots on client
Examples:
# the first phase of putObject
$ gnfd  create-obj test.file gnfd://bucketname/objectname`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  secondarySPFlagName,
				Value: "",
				Usage: "indicate the Secondary SP addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  contentTypeFlagName,
				Value: "",
				Usage: "indicate object content-type",
			},
			&cli.GenericFlag{
				Name: visibilityFlagName,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: inheritType,
				},
				Usage: "set visibility of the object",
			},
		},
	}
}

// cmdPutObj return the command to finish uploading payload of the object
func cmdPutObj() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    uploadObject,
		Usage:     "upload an object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Upload the payload and send with txn to storage provider

Examples:
# the second phase of putObject: upload file to storage provider
$ gnfd put --txnhash xx  file.txt gnfd://bucket-name/file.txt`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     txnHashFlagName,
				Value:    "",
				Usage:    "the txn hash of transaction of createObjectMsg",
				Required: true,
			},
			&cli.StringFlag{
				Name:  contentTypeFlagName,
				Value: "",
				Usage: "indicate object content-type",
			},
		},
	}
}

// createObject get approval of uploading from sp
func createObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErr(fmt.Errorf("args number should be two"))
	}

	urlInfo := ctx.Args().Get(1)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return nil
	}

	// read the local file payload
	filePath := ctx.Args().Get(0)
	exists, objectSize, err := pathExists(filePath)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("upload file not exists")
	} else if objectSize > int64(maxFileSize) {
		return fmt.Errorf("upload file larger than 5G ")
	}

	// Open the referenced file.
	fileReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	gnfdClient, err := NewClient(ctx)
	if err != nil {
		return err
	}

	start := time.Now()
	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	contentType := ctx.String(contentTypeFlagName)
	secondarySPAccs := ctx.String(secondarySPFlagName)

	opts := gnfdclient.CreateObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}

	visibity := ctx.Generic(visibilityFlagName)
	if visibity != "" {
		visibityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibity))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibityTypeVal
	}

	// set second sp address if provided by user
	if secondarySPAccs != "" {
		secondarySplist := strings.Split(secondarySPAccs, ",")
		addrList := make([]sdk.AccAddress, len(secondarySplist))
		for idx, addr := range secondarySplist {
			addrList[idx] = sdk.MustAccAddressFromHex(addr)
		}
		opts.SecondarySPAccs = addrList
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	opts.TxOpts = &txnOpt

	txnHash, err := gnfdClient.CreateObject(c, bucketName, objectName, fileReader, opts)
	if err != nil {
		return err
	}

	fmt.Println("create object successfully, txn hash:", txnHash, "cost time:", time.Since(start).Milliseconds(), "ms")
	return nil
}

// uploadObject upload the payload of file, finish the third stage of putObject
func uploadObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErr(fmt.Errorf("args number more than one"))
	}

	urlInfo := ctx.Args().Get(1)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return nil
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	txnhash := ctx.String(txnHashFlagName)
	// read the local file payload to be uploaded
	filePath := ctx.Args().Get(0)

	exists, objectSize, err := pathExists(filePath)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("upload file not exists")
	} else if objectSize > int64(5*1024*1024*1024) {
		return fmt.Errorf("upload file larger than 5G")
	}

	// Open the referenced file.
	fileReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	opt := spClient.PutObjectOption{}
	contentType := ctx.String(contentTypeFlagName)
	if contentType != "" {
		opt.ContentType = contentType
	}

	if err = client.PutObject(c, bucketName, objectName,
		txnhash, objectSize, fileReader, opt); err != nil {
		fmt.Println("upload object fail:", err.Error())
		return err
	}

	fmt.Printf("upload object: %s successfully ", objectName)
	return nil
}

func pathExists(path string) (bool, int64, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, 0, nil
	}

	if err == nil {
		if stat.IsDir() {
			return false, 0, fmt.Errorf("not support upload dir")
		}
		return true, stat.Size(), nil
	}

	return false, 0, err
}

func getObjAndBucketNames(urlInfo string) (string, string, error) {
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if bucketName == "" || objectName == "" || err != nil {
		return "", "", fmt.Errorf("fail to parse bucket name or object name")
	}
	return bucketName, objectName, nil
}
