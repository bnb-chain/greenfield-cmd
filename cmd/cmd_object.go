package main

import (
	"context"
	"cosmossdk.io/math"
	"errors"
	"fmt"
	"github.com/bnb-chain/greenfield/sdk/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdPutObj return the command to finish uploading payload of the object
func cmdPutObj() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    putObject,
		Usage:     "create object on chain and upload payload of object to SP",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Send createObject txn to chain and upload the payload of object to the storage provider.
The command need to pass the file path inorder to compute hash roots on client

Examples:
# create object and upload file to storage provider, the corresponding object is gnfd-object
$ gnfd-cmd object put file.txt gnfd://gnfd-bucket/gnfd-object`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  secondarySPFlag,
				Value: "",
				Usage: "indicate the Secondary SP addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  contentTypeFlag,
				Value: "",
				Usage: "indicate object content-type",
			},
			&cli.GenericFlag{
				Name: visibilityFlag,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: inheritType,
				},
				Usage: "set visibility of the object",
			},
			&cli.StringFlag{
				Name:  folderFlag,
				Value: "",
				Usage: "indicate folder in bucket to which the object will be uploaded",
			},
		},
	}
}

// cmdCreateFolder create a folder in bucket
func cmdCreateFolder() *cli.Command {
	return &cli.Command{
		Name:      "create-folder",
		Action:    createFolder,
		Usage:     "create a folder in bucket",
		ArgsUsage: " OBJECT-URL ",
		Description: `
Create a folder in bucket, you can set the prefix of folder by --prefix.
Notice that folder is actually an special object.

Examples:
# create folder called gnfd-folder
$ gnfd-cmd object create-folder gnfd://gnfd-bucket/gnfd-folder`,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name: visibilityFlag,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: inheritType,
				},
				Usage: "set visibility of the object",
			},
			&cli.StringFlag{
				Name:  objectPrefix,
				Value: "",
				Usage: "The prefix of the folder to be created",
			},
		},
	}
}

// cmdGetObj return the command to finish downloading object payload
func cmdGetObj() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Action:    getObject,
		Usage:     "download an object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Download a specific object from storage provider

Examples:
# download an object payload to file
$ gnfd-cmd object get gnfd://gnfd-bucket/gnfd-object  file.txt `,
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:  startOffsetFlag,
				Value: 0,
				Usage: "start offset info of the download body",
			},
			&cli.Int64Flag{
				Name:  endOffsetFlag,
				Value: 0,
				Usage: "end offset info of the download body",
			},
		},
	}
}

// cmdCancelObjects cancel the object which has been created
func cmdCancelObjects() *cli.Command {
	return &cli.Command{
		Name:      "cancel",
		Action:    cancelCreateObject,
		Usage:     "cancel the created object",
		ArgsUsage: "OBJECT-URL",
		Description: `
Cancel the created object 

Examples:
$ gnfd-cmd object cancel  gnfd://gnfd-bucket/gnfd-object`,
	}
}

// cmdListObjects list the objects of the bucket
func cmdListObjects() *cli.Command {
	return &cli.Command{
		Name:      "ls",
		Action:    listObjects,
		Usage:     "list objects of the bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
List Objects of the bucket, including object name, object id, object status

Examples:
$ gnfd-cmd object ls gnfd://gnfd-bucket`,
	}
}

// cmdUpdateObject update the visibility of the object
func cmdUpdateObject() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Action:    updateObject,
		Usage:     "update object visibility",
		ArgsUsage: "OBJECT-URL",
		Description: `
Update the visibility of the object.
The visibility value can be public-read, private or inherit.

Examples:
update visibility of the gnfd-object
$ gnfd-cmd object update --visibility=public-read  gnfd://gnfd-bucket/gnfd-object`,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name: visibilityFlag,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: privateType,
				},
				Usage: "set visibility of the bucket",
			},
		},
	}
}

// cmdGetUploadProgress return the uploading progress info of the object
func cmdGetUploadProgress() *cli.Command {
	return &cli.Command{
		Name:      "get-progress",
		Action:    getUploadInfo,
		Usage:     "get the uploading progress info of object",
		ArgsUsage: "OBJECT-URL",
		Description: `
The command is used to get the uploading progress info. 
you can use this command to view the progress information during the process of uploading a file to a Storage Provider.

Examples:
$ gnfd-cmd object get-progress gnfd://gnfd-bucket/gnfd-object`,
	}
}

func cmdMirrorObject() *cli.Command {
	return &cli.Command{
		Name:      "mirror",
		Action:    mirrorObject,
		Usage:     "mirror object to BSC",
		ArgsUsage: "",
		Description: `
Mirror a object as NFT to BSC

Examples:
# Mirror a object using object id
$ gnfd-cmd object mirror --id 1

# Mirror a object using bucket and object name
$ gnfd-cmd object mirror --bucketName yourBucketName --objectName yourObjectName
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     IdFlag,
				Value:    "",
				Usage:    "object id",
				Required: false,
			},
			&cli.StringFlag{
				Name:     bucketNameFlag,
				Value:    "",
				Usage:    "bucket name",
				Required: false,
			},
			&cli.StringFlag{
				Name:     objectNameFlag,
				Value:    "",
				Usage:    "object name",
				Required: false,
			},
		},
	}
}

// putObject upload the payload of file, finish the third stage of putObject
func putObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErr(fmt.Errorf("args number should be 2"))
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
		return fmt.Errorf("upload file larger than 2G ")
	}

	// Open the referenced file.
	fileReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	urlInfo := ctx.Args().Get(1)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		bucketName = ParseBucket(urlInfo)
		if bucketName == "" {
			return toCmdErr(errors.New("fail to parse bucket name"))
		}
		// if the object name has not been set, set the file name as object name
		objectName = filepath.Base(filePath)
	}

	gnfdClient, err := NewClient(ctx)
	if err != nil {
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

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

	folderName := ctx.String(folderFlag)
	if folderName != "" {
		objectName = folderName + "/" + objectName
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

	txnHash, err := gnfdClient.CreateObject(c, bucketName, objectName, fileReader, opts)
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(gnfdClient, c, txnHash, "CreateObject")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("create object %s on chain finish, txn Hash: %s\n", objectName, txnHash)

	if objectSize == 0 {
		return nil
	}

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

	if err = gnfdClient.PutObject(c, bucketName, objectName,
		objectSize, reader, opt); err != nil {
		fmt.Println("put object fail:", err.Error())
		return nil
	}

	// Check if object is sealed
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-timeout:
			return toCmdErr(errors.New("object not sealed after 15 seconds"))
		case <-ticker.C:
			headObjOutput, err := gnfdClient.HeadObject(c, bucketName, objectName)
			if err != nil {
				return err
			}

			if headObjOutput.GetObjectStatus().String() == "OBJECT_STATUS_SEALED" {
				ticker.Stop()
				fmt.Printf("put object %s successfully \n", objectName)
				return nil
			}
		}
	}

}

// getObject download the object payload from sp
func getObject(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return toCmdErr(fmt.Errorf("args number less than one"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	gnfdClient, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelGetObject := context.WithCancel(globalContext)
	defer cancelGetObject()

	_, err = gnfdClient.HeadObject(c, bucketName, objectName)
	if err != nil {
		return toCmdErr(ErrObjectNotExist)
	}

	var filePath string
	if ctx.Args().Len() == 1 {
		filePath = objectName
	} else if ctx.Args().Len() == 2 {
		filePath = ctx.Args().Get(1)
		stat, err := os.Stat(filePath)
		if err == nil {
			if stat.IsDir() {
				if strings.HasSuffix(filePath, "/") {
					filePath += objectName
				} else {
					filePath = filePath + "/" + objectName
				}
			}
		}
	}

	// If file exist, open it in append mode
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}

	defer fd.Close()

	opt := sdktypes.GetObjectOption{}
	startOffset := ctx.Int64(startOffsetFlag)
	endOffset := ctx.Int64(endOffsetFlag)

	// flag has been set
	if startOffset != 0 || endOffset != 0 {
		if err = opt.SetRange(startOffset, endOffset); err != nil {
			return toCmdErr(err)
		}
	}

	body, info, err := gnfdClient.GetObject(c, bucketName, objectName, opt)
	if err != nil {
		return toCmdErr(err)
	}

	_, err = io.Copy(fd, body)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("download object %s successfully, the file path is %s, content length:%d \n", objectName, filePath, uint64(info.Size))

	return nil
}

// cancelCreateObject cancel the created object on chain
func cancelCreateObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	cli, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCancelCreate := context.WithCancel(globalContext)
	defer cancelCancelCreate()

	_, err = cli.HeadObject(c, bucketName, objectName)
	if err != nil {
		return toCmdErr(ErrObjectNotCreated)
	}

	_, err = cli.CancelCreateObject(c, bucketName, objectName, sdktypes.CancelCreateOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Println("cancel create object:", objectName)
	return nil
}

func listObjects(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	c, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	listObjectsRes, err := client.ListObjects(c, bucketName, sdktypes.ListObjectsOptions{})
	if err != nil {
		return toCmdErr(err)
	}

	if len(listObjectsRes.Objects) == 0 {
		fmt.Println("no objects")
		return nil
	}

	listNum := 0
	for _, object := range listObjectsRes.Objects {
		listNum++
		if listNum > maxListObjects {
			return nil
		}
		info := object.ObjectInfo
		if !object.Removed {
			fmt.Printf("object name: %s , object id:%s, object status:%s \n", info.ObjectName, info.Id, info.ObjectStatus)
		}
	}

	return nil

}

func createFolder(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be 1"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	opts := sdktypes.CreateObjectOptions{}

	visibity := ctx.Generic(visibilityFlag)
	if visibity != "" {
		visibityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibity))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibityTypeVal
	}

	objectName = objectName + "/"
	prefix := ctx.String(objectPrefix)
	if prefix != "" {
		objectName = prefix + "/" + objectName
	}

	txnHash, err := client.CreateFolder(c, bucketName, objectName, opts)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	fmt.Printf("create folder: %s successfully, txnHash is %s \n", objectName, txnHash)
	return nil
}

func updateObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	urlInfo := ctx.Args().First()
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelUpdateObject := context.WithCancel(globalContext)
	defer cancelUpdateObject()

	visibility := ctx.Generic(visibilityFlag)
	if visibility == "" {
		return toCmdErr(fmt.Errorf("visibity must set to be updated"))
	}

	visibilityType, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
	if typeErr != nil {
		return typeErr
	}

	txnHash, err := client.UpdateObjectVisibility(c, bucketName, objectName, visibilityType, sdktypes.UpdateObjectOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		fmt.Println("update object visibility error:", err.Error())
		return nil
	}

	err = waitTxnStatus(client, c, txnHash, "UpdateObject")
	if err != nil {
		return toCmdErr(err)
	}

	objectInfo, err := client.HeadObject(c, bucketName, objectName)
	if err != nil {
		// head fail, no need to print the error
		return nil
	}

	fmt.Printf("update object visibility successfully, latest object visibility:%s\n", objectInfo.GetVisibility().String())
	return nil
}

func getUploadInfo(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be 1"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelGetUploadInfo := context.WithCancel(globalContext)
	defer cancelGetUploadInfo()

	uploadInfo, err := client.GetObjectUploadProgress(c, bucketName, objectName)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Println("uploading progress:", uploadInfo)
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

func mirrorObject(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	id := math.NewUint(0)
	if ctx.String(IdFlag) != "" {
		id = math.NewUintFromString(ctx.String(IdFlag))
	}

	bucketName := ctx.String(bucketNameFlag)
	objectName := ctx.String(objectNameFlag)

	c, cancelContext := context.WithCancel(globalContext)
	defer cancelContext()

	txResp, err := client.MirrorObject(c, id, bucketName, objectName, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("mirror object succ, txHash: %s\n", txResp.TxHash)
	return nil
}
