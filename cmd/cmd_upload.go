package main

import (
	"context"
	"fmt"
	"log"
	"os"

	inscription "github.com/bnb-chain/greenfield-sdk-go"
	"github.com/urfave/cli/v2"
)

// cmdSendPutTxn return the command to finish first stage of putObject
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
$ gnfd put-txn file.txt s3://bucket-name/object-name`,
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
			&cli.StringFlag{
				Name:  "content-hash",
				Value: "",
				Usage: "indicate object sha256 hex hash",
			},
			&cli.IntFlag{
				Name:  "object-size",
				Value: 10,
				Usage: "the object payload size",
			},
		},
	}
}

// cmdPutObj return the command to finish second stage of putObject
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
$ gnfd  pre-upload s3://bucketname/object`,
	}
}

// cmdPutObj return the command to finish second stage of putObject
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
$ gnfd put --txnhash xx  file.txt s3://bucket-name/file.txt`,
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
		log.Fatalln(err.Error())
	}
	defer f.Close()

	sha256hash := inscription.CalcSHA256Hash(f)

	size := int64(ctx.Int("object-size"))
	if size <= 0 {
		size, _ = inscription.GetContentLength(f)
	}

	putObjectMeta := inscription.PutObjectMeta{
		PaymentAccount: s3Client.GetAccount(),
		PrimarySp:      primarySP,
		IsPublic:       isPublic,
		ObjectSize:     size,
		Sha256Hash:     sha256hash,
		ContentType:    contentType,
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	txnInfo, err := s3Client.SendPutObjectTxn(c, bucketName, objectName, putObjectMeta)
	if err != nil {
		fmt.Println("send putObject txn fail", err)
		return err
	}

	fmt.Println("send putObject txn msg succ, got txn hash:", txnInfo.String())
	return nil
}

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
	log.Printf("uploading file:%s, objectName:%s \n", filePath, objectName)

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

	contentSha256, err := computeFileCheckSum(filePath)
	if err != nil {
		return err
	}
	res, err := s3Client.PutObjectWithTxn(c, txnhash, objectName, bucketName, contentSha256, fileReader,
		objectSize, inscription.PutObjectOptions{})

	if err != nil {
		fmt.Println("upload payload fail:", err.Error())
		return err
	}

	fmt.Println("upload succ:", res.String())
	return nil
}

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

	signature, _, err := s3Client.GetApproval(c, bucketName, objectName)
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

func computeFileCheckSum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return inscription.CalcSHA256Hash(f), nil
}

func getObjAndBucketNames(urlInfo string) (string, string, error) {
	bucketName, objectName := ParseBucketAndObject(urlInfo)
	if bucketName == "" || objectName == "" {
		return "", "", fmt.Errorf("fail to parse bucket name or object name")
	}
	return bucketName, objectName, nil
}
