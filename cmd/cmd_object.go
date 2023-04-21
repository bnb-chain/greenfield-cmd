package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	spClient "github.com/bnb-chain/greenfield-go-sdk/client/sp"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"

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
# create object and upload file to storage provider, the corresponding object is gnfdObject
$ gnfd-cmd -c config.toml put file.txt gnfd://gnfdBucket/gnfdObject`,
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
$ gnfd -c config.toml get gnfd://gnfdBucket/gnfdObject  file.txt `,
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:  startOffsetFlagName,
				Value: 0,
				Usage: "start offset info of the download body",
			},
			&cli.Int64Flag{
				Name:  endOffsetFlagName,
				Value: 0,
				Usage: "end offset info of the download body",
			},
		},
	}
}

// cmdCancelObjects cancel the object which has been created
func cmdCancelObjects() *cli.Command {
	return &cli.Command{
		Name:      "cancel-create-obj",
		Action:    cancelCreateObject,
		Usage:     "cancel the created object",
		ArgsUsage: "OBJECT-URL",
		Description: `
Cancel the created object 

Examples:
$ gnfd  cancel-create-obj gnfd://gnfdBucket/gnfdObject`,
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
$ gnfd  ls  gnfd://gnfdBucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  userAddressFlagName,
				Value: "",
				Usage: "indicate which user's buckets to be list, you" +
					" don't need to specify this if you want to list your own bucket ",
			},
		},
	}
}

// cmdPutObjPolicy
func cmdPutObjPolicy() *cli.Command {
	return &cli.Command{
		Name:      "put-obj-policy",
		Action:    putObjectPolicy,
		Usage:     "put object policy to group or account",
		ArgsUsage: " OBJECT-URL",
		Description: `
The command is used to set the object policy of the granted account or group-id.
It required to set granted account or group-id by --groupId or --granter.

Examples:
$ gnfd-cmd -c config.toml put-obj-policy --groupId 111 --action get,delete gnfd://gnfdBucket/gnfdObject`,
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:  groupIDFlagName,
				Value: 0,
				Usage: "the group id of the group",
			},
			&cli.StringFlag{
				Name:  granterFlagName,
				Value: "",
				Usage: "the account address to set the policy",
			},
			&cli.StringFlag{
				Name:  actionsFlagName,
				Value: "",
				Usage: "set the actions of the policy," +
					"actions can be the following: create, delete, copy, get or execute." +
					" multi actions like \"delete,copy\" is supported",
				Required: true,
			},
			&cli.GenericFlag{
				Name: effectFlagName,
				Value: &CmdEnumValue{
					Enum:    []string{effectDeny, effectAllow},
					Default: effectAllow,
				},
				Usage: "set the effect of the policy",
			},
			&cli.Uint64Flag{
				Name:  expireTimeFlagName,
				Value: 0,
				Usage: "set the expire unix time stamp of the policy",
			},
		},
	}
}

// putObject upload the payload of file, finish the third stage of putObject
func putObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErr(fmt.Errorf("args number should be 2"))
	}

	urlInfo := ctx.Args().Get(1)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
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

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	contentType := ctx.String(contentTypeFlagName)
	secondarySPAccs := ctx.String(secondarySPFlagName)

	opts := gnfdclient.CreateObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}

	visibility := ctx.Generic(visibilityFlagName)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
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

	_, err = client.HeadObject(c, bucketName, objectName)
	if err != nil {
		time.Sleep(5 * time.Second)
		_, err = client.HeadObject(c, bucketName, objectName)
		if err != nil {
			return toCmdErr(ErrObjectNotCreated)
		}
	}

	fmt.Printf("create object %s on chain finish \n", objectName)

	opt := spClient.PutObjectOption{}
	if contentType != "" {
		opt.ContentType = contentType
	}
	// Open the referenced file.
	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err = client.PutObject(c, bucketName, objectName,
		txnHash, objectSize, reader, opt); err != nil {
		fmt.Println("put object fail:", err.Error())
		return nil
	}

	fmt.Printf("put object: %s successfully \n", objectName)
	return nil
}

func putObjectPolicy(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}
	urlInfo := ctx.Args().Get(0)

	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	groupId := ctx.Uint64(groupIDFlagName)
	granter := ctx.String(granterFlagName)
	principal, err := parsePrincipal(ctx, granter, groupId)
	if err != nil {
		return toCmdErr(err)
	}

	actions, err := parseActions(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	effect := permTypes.EFFECT_ALLOW
	effectStr := ctx.String(effectFlagName)
	if effectStr != "" {
		if effectStr == effectDeny {
			effect = permTypes.EFFECT_DENY
		}
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	expireTime := ctx.Uint64(expireTimeFlagName)
	var statement permTypes.Statement
	if expireTime > 0 {
		tm := time.Unix(int64(expireTime), 0)
		statement = gnfdclient.NewStatement(actions, effect, nil, gnfdclient.NewStatementOptions{StatementExpireTime: &tm})
	} else {
		statement = gnfdclient.NewStatement(actions, effect, nil, gnfdclient.NewStatementOptions{})
	}
	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txOpts := &types.TxOption{Mode: &broadcastMode}

	statements := []*permTypes.Statement{&statement}
	policyTx, err := client.PutObjectPolicy(bucketName, objectName, principal, statements,
		gnfdclient.PutPolicyOption{TxOpts: txOpts})

	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("put object policy %s succ, txn hash: %s\n", bucketName, policyTx)

	c, cancelPutPolicy := context.WithCancel(globalContext)
	defer cancelPutPolicy()

	// get the latest policy from chain
	if groupId > 0 {
		policyInfo, err := client.GetObjectPolicyOfGroup(c, bucketName, objectName, groupId)
		if err == nil {
			fmt.Printf("policy info of the group: \n %s\n", policyInfo.String())
		}
	} else {
		granterAddr, err := sdk.AccAddressFromHexUnsafe(granter)
		if err == nil {
			policyInfo, err := client.GetObjectPolicy(c, bucketName, objectName, granterAddr)
			if err == nil {
				fmt.Printf("policy info of the account:  \n %s\n", policyInfo.String())
			}
		}
	}

	return nil
}

// getObject download the object payload from sp
func getObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErr(fmt.Errorf("args number more than one"))
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

	filePath := ctx.Args().Get(1)

	// If file exist, open it in append mode
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	defer fd.Close()

	opt := spClient.GetObjectOption{}
	startOffset := ctx.Int64(endOffsetFlagName)
	endOffset := ctx.Int64(endOffsetFlagName)

	// flag has been set
	if startOffset != 0 || endOffset != 0 {
		if err = opt.SetRange(startOffset, endOffset); err != nil {
			return toCmdErr(err)
		}
	}

	body, _, err := gnfdClient.GetObject(c, bucketName, objectName, opt)
	if err != nil {
		return toCmdErr(err)
	}

	_, err = io.Copy(fd, body)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("download object %s successfully, the file path is %s,", objectName, filePath)

	return nil
}

// cancelCreateObject
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

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}

	_, err = cli.CancelCreateObject(bucketName, objectName, gnfdclient.CancelCreateOption{TxOpts: &txnOpt})
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
	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	listObjectsRes, err := client.SPClient.ListObjects(c, bucketName, spClient.NewAuthInfo(false, ""))

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
