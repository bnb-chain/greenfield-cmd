package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// ===================== sub commands ================

// cmdListObjects list the objects of the bucket
func cmdS3ListBuckets() *cli.Command {
	return &cli.Command{
		Name:      "s3-ls-bucket",
		Action:    S3ListBuckets,
		Usage:     "list s3 buckets in specific region",
		ArgsUsage: "BUCKET-REGION",
		Description: `
List the s3 buckets in specific region

Examples:
$ gnfd-cmd storage s3-ls-bucket ap-northeast-2`,
	}
}

// cmdListObjects list the objects of the bucket
func cmdS3ListObjects() *cli.Command {
	return &cli.Command{
		Name:      "s3-ls",
		Action:    S3ListObjects,
		Usage:     "list objects of the aws s3 bucket in specific region",
		ArgsUsage: "[regionName] BUCKET-NAME",
		Description: `
List Objects of the s3 bucket, including object name, object id, object status

Examples:
# query the object in the bucket
$ gnfd  s3-ls ap-northeast-2 myawsbucket`,
	}
}

// cmdListObjects list the objects of the bucket
func cmdS3DownloadObjects() *cli.Command {
	return &cli.Command{
		Name:      "s3-download-objects",
		Action:    S3DownloadObjects,
		Usage:     "download all objects in the aws s3 bucket",
		ArgsUsage: "[regionName] BUCKET-NAME DOWNLOAD-DIR",
		Description: `
Download All Objects in the aws s3 bucket

Examples:
# download objects 
$ gnfd  s3-ls ap-northeast-2 myawsbucket ./downloads`,
	}
}

// cmdPutObj return the command to finish uploading payload of the object
func cmdS3MigrationObjects() *cli.Command {
	return &cli.Command{
		Name:      "s3-migration-objects",
		Action:    S3MigrationObjects,
		Usage:     "migration all object in the aws specific bucket",
		ArgsUsage: "[regionName] BUCKET-NAME GREENFIELD-BUCKET-NAME",
		Description: `
Migrate All Objects in the specific aws s3 bucket

Examples:
$ gnfd-cmd storage s3-migration-objects ap-northeast-2 before-bucket after-bucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  secondarySPFlag,
				Value: "",
				Usage: "indicate the Secondary SP addr string list, input like addr1,addr2,addr3",
			},
			&cli.GenericFlag{
				Name: visibilityFlag,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: inheritType,
				},
				Usage: "set visibility of the object",
			},
		},
	}
}

// ===================== internal functions ================

// cmdListObjects list the objects of the bucket
func S3ListBuckets(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErrWithContext(ctx, fmt.Errorf("args number should be one"))
	}

	regionName := ctx.Args().Get(0)

	s3Client, err := NewS3Client(ctx, regionName)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	resp, err := s3Client.ListBucketsWithContext(c, nil)
	if err != nil {
		return toCmdErr(err)
	}

	if len(resp.Buckets) == 0 {
		fmt.Printf("no buckets in %s\n", regionName)
		return nil
	}

	fmt.Printf("Oh!, you have %d buckets in %s\n", len(resp.Buckets), regionName)
	for idx, bucket := range resp.Buckets {
		fmt.Printf("%d) found a bucket: %v\n", idx, bucket)
	}

	return nil
}

// cmdListObjects list the objects of the bucket
func S3ListObjects(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErrWithContext(ctx, fmt.Errorf("args number should be two"))
	}

	regionName := ctx.Args().Get(0)
	bucketName := ctx.Args().Get(1)

	s3Client, err := NewS3Client(ctx, regionName)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	// Get a list of objects in the bucket
	resp, err := s3Client.ListObjectsV2WithContext(c, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("incorrect bucket name in %s region, please check your bucket name\n", regionName)
	}

	if len(resp.Contents) == 0 {
		fmt.Printf("there is no any objects in %s bucket\n", bucketName)
		return nil
	}

	// Recursively exploring objects in a bucket
	for _, object := range resp.Contents {

		// check it is dir, not a file
		// in AWS S3 dir Size is 0, so we can filter by size
		if *object.Size == 0 {
			fmt.Println("found a dir, it's skipped.")
			continue
		} else {
			// only file object
			if strings.Contains(*object.Key, "/") {
				// fmt.Println("this content is in another depth")
				fmt.Println("found a object", object)
				continue
			}

			fmt.Println("found a object", object)
		}
		fmt.Println()
	}

	return nil
}

func S3DownloadObjects(ctx *cli.Context) error {
	if ctx.NArg() != 3 {
		return toCmdErrWithContext(ctx, fmt.Errorf("args number should be three"))
	}

	regionName := ctx.Args().Get(0)
	bucketName := ctx.Args().Get(1)
	downloadPath := ctx.Args().Get(2)

	isDirExist, _, _ := dirExists(downloadPath)
	if !isDirExist {
		if err := os.Mkdir(downloadPath, os.ModePerm); err != nil {
			return toCmdErr(fmt.Errorf("the download directory is not exisited and failed to create new dir"))
		}
		fmt.Println("made a new directory")
	}

	s3Client, err := NewS3Client(ctx, regionName)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	// Get a list of objects in the bucket
	resp, err := s3Client.ListObjectsV2WithContext(c, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("incorrect bucket name in %s region, please check your bucket name\n", regionName)
	}

	if len(resp.Contents) == 0 {
		fmt.Printf("there is no any objects in %s bucket\n", bucketName)
		return nil
	}

	// Recursively exploring objects in a bucket
	for _, object := range resp.Contents {

		// check it is dir, not a file
		// in AWS S3 dir Size is 0, so we can filter by size
		if *object.Size == 0 {
			fmt.Println("found a dir, it's skipped.")
			fmt.Println()
			continue
		} else {
			// only file object
			if strings.Contains(*object.Key, "/") {
				// fmt.Println("this content is in another depth")
				_, err := downloadObject(s3Client, bucketName, *object.Key, downloadPath)
				if err != nil {
					fmt.Println("failed to download a object: ", *object.Key)
					return err
				}
				fmt.Println()
				continue
			}
			fmt.Println("Downloading a object: ", *object.Key)
			_, err := downloadObject(s3Client, bucketName, *object.Key, downloadPath)
			if err != nil {
				fmt.Println("failed to download a object: ", *object.Key)
				return err
			}
		}
		fmt.Println()
	}
	return nil
}

// gnfd-cmd storage put ./data/genesis.json gnfd://testbucket/testobject
// putObject upload the payload of file, finish the third stage of putObject
func S3MigrationObjects(ctx *cli.Context) error {
	if ctx.NArg() != 3 {
		return toCmdErrWithContext(ctx, fmt.Errorf("args number should be two"))
	}

	regionName := ctx.Args().Get(0)
	bucketName := ctx.Args().Get(1)
	greenFieldBucketName := ctx.Args().Get(2)
	tempDownloadPath := "./tmp"

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()
	// check
	// the bucket is exisited?
	_ = greenFieldBucketName

	greenfieldClient, err := NewClient(ctx)
	if err != nil {
		return err
	}

	// rm -rf all content ./tmp/*
	isDirExist, _, _ := dirExists(tempDownloadPath)
	if !isDirExist {
		if err := os.Mkdir(tempDownloadPath, os.ModePerm); err != nil {
			return toCmdErr(fmt.Errorf("the download directory is not exisited and failed to create new dir"))
		}
		fmt.Println("made a new directory")
	}

	s3Client, err := NewS3Client(ctx, regionName)
	if err != nil {
		return toCmdErr(err)
	}

	// Get a list of objects in the bucket
	resp, err := s3Client.ListObjectsV2WithContext(c, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("incorrect bucket name in %s region, please check your bucket name\n", regionName)
	}

	if len(resp.Contents) == 0 {
		fmt.Printf("there is no any objects in %s bucket\n", bucketName)
		return nil
	}

	// Recursively exploring objects in a bucket
	for _, object := range resp.Contents {
		// check it is dir, not a file
		// in AWS S3 dir Size is 0, so we can filter by size
		if *object.Size == 0 {
			fmt.Println("found a dir, it's skipped.")
			fmt.Println()
			continue
		} else {
			// only file object
			if strings.Contains(*object.Key, "/") {
				// fmt.Println("this content is in another depth")
				_, err := downloadObject(s3Client, bucketName, *object.Key, tempDownloadPath)
				if err != nil {
					fmt.Println("failed to download a object: ", *object.Key)
					return err
				}
				fmt.Println()
				continue
			}
			fmt.Println("Downloading a object: ", *object.Key)
			_, err := downloadObject(s3Client, bucketName, *object.Key, tempDownloadPath)
			if err != nil {
				fmt.Println("failed to download a object: ", *object.Key)
				return err
			}
		}
		fmt.Println()
	}

	// scanning files in temp dir
	files, err := ioutil.ReadDir("./tmp/")
	if err != nil {
		return fmt.Errorf("failed read tmp dir")
	}

	// creating txs and uploading files after checking file statstics
	for _, file := range files {
		objectName := file.Name()
		filePath := tempDownloadPath + "/" + file.Name()
		fmt.Println("Uploading a object:", objectName)

		stat, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat the file: %s\n", file.Name())
		}

		objectSize := stat.Size()
		if objectSize > int64(maxFileSize) {
			return fmt.Errorf("failed to upload, the file size is larger than 5G")
		}

		// Open the referenced file.
		fileReader, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer fileReader.Close()

		contentType := ctx.String(contentTypeFlag)
		secondarySPAccs := ctx.String(secondarySPFlag)
		opts := sdktypes.CreateObjectOptions{}
		if contentType != "" {
			opts.ContentType = contentType
		}

		visibity := ctx.Generic(visibilityFlag)
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

		c, cancelList := context.WithCancel(globalContext)
		defer cancelList()

		// create txn for DA
		txnHash, err := greenfieldClient.CreateObject(c, greenFieldBucketName, objectName, fileReader, opts)
		if err != nil {
			fmt.Errorf("failed to create object: %s\n", err.Error())
			continue
		}

		fmt.Printf("create object %s on chain finish, txn Hash: %s\n", objectName, txnHash)
		fmt.Printf("you can find the tx on there, %s\n", fmt.Sprintf("https://greenfieldscan.com/tx/%s", txnHash))

		// upload file at SP
		opt := sdktypes.PutObjectOptions{}
		if contentType != "" {
			opt.ContentType = contentType
		}
		opt.TxnHash = txnHash

		// Open the referenced file.
		reader, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer reader.Close()

		if err = greenfieldClient.PutObject(c, greenFieldBucketName, objectName,
			objectSize, reader, opt); err != nil {
			fmt.Println("put object fail:", err.Error())
			removeTmpFiles(filePath)
			return nil
		}

		// Check if object is sealed
		time.Sleep(5 * time.Second)
		headObjOutput, err := greenfieldClient.HeadObject(c, greenFieldBucketName, objectName)
		if err != nil {
			return err
		}

		if headObjOutput.GetObjectStatus().String() == "OBJECT_STATUS_SEALED" {
			fmt.Printf("put object %s successfully \n", objectName)
			fmt.Printf("you can find the tx on there, %s\n", fmt.Sprintf("https://greenfieldscan.com/tx/%s", txnHash))

			removeTmpFiles(filePath)
			fmt.Println("remove uploaded file:", filePath)
		}

	}

	removeTmpFiles(tempDownloadPath)
	return nil
}

// ===================utility function =================
func removeTmpFiles(tempDownloadPath string) {
	// Remove all the directories and files
	// Using RemoveAll() function
	err := os.RemoveAll(tempDownloadPath)
	if err != nil {
		fmt.Println("failed to remove temporary files in temp dir")
		return
	}
	return
}

func downloadObject(s3Client *s3.S3, bucketName, key, downloadPath string) (string, error) {
	var filePath string
	downloader := s3manager.NewDownloaderWithClient(s3Client)
	filePath = filepath.Join(downloadPath, filepath.Base(key))

	if strings.Contains(key, "/") {
		slice := strings.Split(key, "/")
		path := slice[len(slice)-1]
		filePath = filepath.Join(downloadPath, filepath.Base(path))
	}

	// create a file and open
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}

	fmt.Println("Downloaded:", key)
	return filePath, nil
}
