package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	spClient "github.com/bnb-chain/greenfield-go-sdk/client/sp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/urfave/cli/v2"
)

// cmdPreCreateObj send the request get approval of uploading
func cmdPreCreateObj() *cli.Command {
	return &cli.Command{
		Name:      "create-obj",
		Action:    createObject,
		Usage:     "create object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Get approval from storage provider and send createObject txn to chain.
The command need to pass the file path inorder to compute hash roots on client
Examples:
# the first phase of putObject
$ gnfd  create-obj test.file gnfd://bucketname/objectname`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "public",
				Value: false,
				Usage: "indicate whether the object is public",
			},
			&cli.StringFlag{
				Name:  "SecondarySPs",
				Value: "",
				Usage: "indicate the Secondary SP addr, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  "contentType",
				Value: "application/xml",
				Usage: "indicate object content-type",
			},
		},
	}
}

// cmdPutObj return the command to finish uploading payload of the object
func cmdPutObj() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    uploadObject,
		Usage:     "upload object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Upload the payload and send with txn to storage provider

Examples:
# the second phase of putObject: upload file to storage provider
$ gnfd put --txnhash xx  file.txt gnfd://bucket-name/file.txt`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "txnhash",
				Value:    "",
				Usage:    "the txn hash of transaction of createObjectMsg",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "content-type",
				Value: "application/xml",
				Usage: "indicate object content-type",
			},
		},
	}
}

// uploadObject upload the payload of file, finish the third stage of putObject
func uploadObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("the args number should be two")
	}

	urlInfo := ctx.Args().Get(1)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return nil
	}

	s3Client, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	txnhash := ctx.String("txnhash")
	// read the local file payload to be uploaded
	filePath := ctx.Args().Get(0)

	exists, objectSize, err := pathExists(filePath)
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

	opt := spClient.UploadOption{}
	contentType := ctx.String("content-type")
	if contentType != "" {
		opt.ContentType = contentType
	}

	res, err := s3Client.UploadObject(c, bucketName, objectName, txnhash, objectSize, fileReader, opt)

	if err != nil {
		fmt.Println("upload object fail:", err.Error())
		return err
	}

	fmt.Println("upload object succ:", res.String())
	return nil
}

// createObject get approval of uploading from sp
func createObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("the args number should be two")
	}

	urlInfo := ctx.Args().Get(1)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return nil
	}

	// read the local file payload
	filePath := ctx.Args().Get(0)
	exists, objectSize, err := pathExists(filePath)
	if !exists {
		return fmt.Errorf("upload file not exists")
	} else if objectSize > int64(5*1024*1024*1024) {
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
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	isPublic := ctx.Bool("public")
	contentType := ctx.String("contentType")
	secondarySPAccs := ctx.String("SecondarySPs")

	opts := gnfdclient.CreateObjectOptions{}
	opts.IsPublic = isPublic
	if contentType != "" {
		opts.ContentType = contentType
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

	gnfdResp := gnfdClient.CreateObject(c, bucketName, objectName, fileReader, opts)
	if gnfdResp.Err != nil {
		fmt.Println("create object fail:", gnfdResp.Err.Error())
		return err
	}

	fmt.Println("createObject txn hash:", gnfdResp.TxnHash)
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
	bucketName, objectName := ParseBucketAndObject(urlInfo)
	if bucketName == "" || objectName == "" {
		return "", "", fmt.Errorf("fail to parse bucket name or object name")
	}
	return bucketName, objectName, nil
}
