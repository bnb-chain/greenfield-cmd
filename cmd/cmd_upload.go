package main

import (
	"context"
	"fmt"
	"log"
	"os"

	greenfield "github.com/bnb-chain/greenfield-sdk-go"
	"github.com/bnb-chain/greenfield-sdk-go/pkg/signer"
	"github.com/urfave/cli/v2"
)

// cmdSendPutTxn finish first stage of putObject command
func cmdSendPutTxn() *cli.Command {
	return &cli.Command{
		Name:      "put-txn",
		Action:    sendPutTxn,
		Usage:     "create a new object on chain",
		ArgsUsage: "OBJECT-URL",
		Description: `
send a createObjMsg to storage provider

Examples:
# the first phase of putObject: send putObjectMsg to SP, will not upload payload
$ gnfd put-txn file.txt gnfd://bucket-name/object-name`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "public",
				Value:    false,
				Usage:    "indicate whether the object is public",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "primary-sp",
				Value: "sp-1",
				Usage: "indicate the primary sp id",
			},
			&cli.StringFlag{
				Name:     "content-type",
				Value:    "application/xml",
				Usage:    "indicate object content-type",
				Required: true,
			},
		},
	}
}

// cmdPreCreateObj send the request get approval of uploading
func cmdPreCreateObj() *cli.Command {
	return &cli.Command{
		Name:      "pre-upload",
		Action:    preUploadObject,
		Usage:     "pre create object",
		ArgsUsage: "OBJECT-URL",
		Description: `
 preUpload and get approval from storage provider

Examples:
# the first phase of putObject
$ gnfd  pre-upload gnfd://bucketname/object`,
	}
}

// cmdPutObj return the command to finish uploading payload of the object
func cmdPutObj() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    uploadObject,
		Usage:     "upload object payload",
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
				Usage:    "the txn hash of tranction of createObjectMsg",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "filepath",
				Value: "",
				Usage: "file path info to be uploaded",
			},
		},
	}
}

// sendPutTxn send to request of create object chain message,
// it finishes the first stage of putObject
// TODO(leo) conbine it with PreObject
func sendPutTxn(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("the args should contain s3-url and filePath")
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

	filePath := ctx.Args().Get(0)
	log.Printf("uploading file:%s, objectName:%s \n", filePath, objectName)

	isPublic := ctx.Bool("public")
	primarySP := ctx.String("primary-sp")
	contentType := ctx.String("content-type")

	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	size := int64(ctx.Int("object-size"))
	if size <= 0 {
		size, _ = greenfield.GetContentLength(f)
	}

	putObjectMeta := greenfield.PutObjectMeta{
		PaymentAccount: s3Client.GetAccount(),
		PrimarySp:      primarySP,
		IsPublic:       isPublic,
		ObjectSize:     size,
		ContentType:    contentType,
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	txnHash, err := s3Client.PrePutObject(c, bucketName, objectName, putObjectMeta, f,
		signer.NewAuthInfo(false, ""))

	if err != nil {
		fmt.Println("send putObject txn fail", err)
		return err
	}

	fmt.Println("send putObject txn msg succ, got txn hash:", txnHash)
	return nil
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
	} else if objectSize > int64(500*1024*1024) {
		return fmt.Errorf("upload file larger than 500M ")
	}

	// Open the referenced file.
	fileReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	meta := greenfield.ObjectMeta{
		ObjectSize:  objectSize,
		ContentType: "application/octet-stream",
	}

	res, err := s3Client.PutObject(c, bucketName, objectName, fileReader, txnhash, meta, signer.NewAuthInfo(false, ""))

	if err != nil {
		fmt.Println("upload payload fail:", err.Error())
		return err
	}

	fmt.Println("upload succ:", res.String())
	return nil
}

// preUploadObject get approval of uploading from sp
func preUploadObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("the args number should be two")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return nil
	}
	s3Client, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	signature, err := s3Client.GetApproval(c, bucketName, objectName, signer.NewAuthInfo(false, ""))
	if err != nil {
		return err
	}

	fmt.Printf("get signature:%s\n", signature)
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
