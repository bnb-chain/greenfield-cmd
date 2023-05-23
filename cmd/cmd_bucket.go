package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bnb-chain/greenfield-go-sdk/pkg/utils"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdCreateBucket create a new Bucket
func cmdCreateBucket() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Action:    createBucket,
		Usage:     "create a new bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Create a new bucket and set a createBucketMsg to storage provider.
The bucket name should unique and the default visibility is private.
The command need to set the primary SP address with --primarySP.

Examples:
# Create a new bucket called gnfd-bucket, visibility is public-read
$ gnfd-cmd bucket create --visibility=public-read  gnfd://gnfd-bucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  primarySPFlag,
				Value: "",
				Usage: "indicate the primarySP address, using the string type",
			},
			&cli.StringFlag{
				Name:  paymentFlag,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlag,
				Value: 0,
				Usage: "indicate the read quota info of the bucket",
			},
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

// cmdUpdateBucket create a new Bucket
func cmdUpdateBucket() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Action:    updateBucket,
		Usage:     "update bucket meta on chain",
		ArgsUsage: "BUCKET-URL",
		Description: `
Update the visibility, payment account or read quota meta of the bucket.
The visibility value can be public-read, private or inherit.
You can update only one item or multiple items at the same time.

Examples:
update visibility and the payment address of the gnfd-bucket
$ gnfd-cmd bucket update --visibility=public-read --paymentAddress xx  gnfd://gnfd-bucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  paymentFlag,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlag,
				Usage: "indicate the read quota info of the bucket",
			},
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

// cmdListBuckets list the bucket of the owner
func cmdListBuckets() *cli.Command {
	return &cli.Command{
		Name:      "ls",
		Action:    listBuckets,
		Usage:     "list buckets",
		ArgsUsage: "",
		Description: `
List the bucket names and bucket ids of the user.

Examples:
$ gnfd-cmd bucket ls`,
	}
}

func cmdPutBucketPolicy() *cli.Command {
	return &cli.Command{
		Name:      "put-bucket-policy",
		Action:    putBucketPolicy,
		Usage:     "put bucket policy to group or account",
		ArgsUsage: " BUCKET-URL",
		Description: `
The command is used to set the bucket policy of the grantee account or group-id.
It required to set the grantee account or group-id by --grantee or --groupId.

Examples:
$ gnfd-cmd put-bucket-policy --groupId 111 --actions delete,update gnfd://gnfd-bucket/gnfd-object`,
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:  groupIDFlag,
				Value: 0,
				Usage: "the group id of the group",
			},
			&cli.StringFlag{
				Name:  granteeFlag,
				Value: "",
				Usage: "the address hex string of the grantee",
			},
			&cli.StringFlag{
				Name:  actionsFlag,
				Value: "",
				Usage: "set the actions of the policy," +
					"actions can be the following: delete, update, deleteObj, copyObj, getObj, executeObj, list or all" +
					", multi actions like \"delete,update\" is supported," +
					" the actions which contain Obj means it is a action for the objects in the bucket, for example," +
					" the deleteObj means grant the permission of delete Objects in the bucket",
				Required: true,
			},
			&cli.GenericFlag{
				Name: effectFlag,
				Value: &CmdEnumValue{
					Enum:    []string{effectDeny, effectAllow},
					Default: effectAllow,
				},
				Usage: "set the effect of the policy",
			},
			&cli.Uint64Flag{
				Name:  expireTimeFlag,
				Value: 0,
				Usage: "set the expire unix time stamp of the policy",
			},
		},
	}
}

// createBucket send the create bucket request to storage provider
func createBucket(ctx *cli.Context) error {
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

	primarySpAddrStr := ctx.String(primarySPFlag)
	if primarySpAddrStr == "" {
		// if primarySP not set, choose sp0 as the primary sp
		spInfo, err := client.ListStorageProviders(c, false)
		if err != nil {
			return toCmdErr(errors.New("fail to get primary sp address"))
		}
		primarySpAddrStr = spInfo[0].GetOperatorAddress()
		fmt.Println("choose primary sp:", spInfo[0].GetEndpoint())
	}

	opts := sdktypes.CreateBucketOptions{}
	paymentAddrStr := ctx.String(paymentFlag)
	if paymentAddrStr != "" {
		opts.PaymentAddress = paymentAddrStr
	}

	visibility := ctx.Generic(visibilityFlag)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlag)
	if chargedQuota > 0 {
		opts.ChargedQuota = chargedQuota
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	opts.TxOpts = &types.TxOption{Mode: &broadcastMode}

	txnHash, err := client.CreateBucket(c, bucketName, primarySpAddrStr, opts)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("create bucket %s succ, txn hash: %s\n", bucketName, txnHash)
	return nil
}

// updateBucket send the create bucket request to storage provider
func updateBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelUpdateBucket := context.WithCancel(globalContext)
	defer cancelUpdateBucket()

	// if bucket not exist, no need to update it
	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	opts := sdktypes.UpdateBucketOption{}
	paymentAddrStr := ctx.String(paymentFlag)
	if paymentAddrStr != "" {
		opts.PaymentAddress = paymentAddrStr
	}

	visibility := ctx.Generic(visibilityFlag)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlag)
	if chargedQuota > 0 {
		opts.ChargedQuota = &chargedQuota
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	opts.TxOpts = &txnOpt

	_, err = client.UpdateBucketInfo(c, bucketName, opts)
	if err != nil {
		fmt.Println("update bucket error:", err.Error())
		return nil
	}

	bucketInfo, err := client.HeadBucket(c, bucketName)
	if err != nil {
		// head fail, no need to print the error
		return nil
	}

	fmt.Printf("latest bucket meta on chain:\nvisibility:%s\nread quota:%d\npayment address:%s \n", bucketInfo.GetVisibility().String(),
		bucketInfo.GetChargedReadQuota(), bucketInfo.GetPaymentAddress())
	return nil
}

// listBuckets list the buckets of the specific owner
func listBuckets(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	bucketListRes, err := client.ListBuckets(c)

	if err != nil {
		return toCmdErr(err)
	}

	if len(bucketListRes.Buckets) == 0 {
		fmt.Println("no buckets")
		return nil
	}

	fmt.Println("bucket list:")
	for _, bucket := range bucketListRes.Buckets {
		info := bucket.BucketInfo
		if !bucket.Removed {
			fmt.Printf("bucket name: %s, bucket id: %s \n", info.BucketName, info.Id)
		}
	}
	return nil

}

func putBucketPolicy(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)
	principal, err := parsePrincipal(grantee, groupId)
	if err != nil {
		return toCmdErr(err)
	}

	actions, err := parseActions(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	effect := permTypes.EFFECT_ALLOW
	effectStr := ctx.String(effectFlag)
	if effectStr != "" {
		if effectStr == effectDeny {
			effect = permTypes.EFFECT_DENY
		}
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	expireTime := ctx.Uint64(expireTimeFlag)
	var statement permTypes.Statement

	var resources []string
	actionsString := ctx.String(actionsFlag)
	// if the actions is *Object (expect createObject), set the resource to be "grn:o::bucketName/*"
	if (strings.Contains(actionsString, "Obj") || strings.Contains(actionsString, "all")) && actionsString != "create" {
		resources = []string{
			fmt.Sprintf("grn:o::%s/%s", bucketName, "*")}
	}

	if expireTime > 0 {
		tm := time.Unix(int64(expireTime), 0)
		statement = utils.NewStatement(actions, effect, resources, sdktypes.NewStatementOptions{StatementExpireTime: &tm})
	} else {
		statement = utils.NewStatement(actions, effect, resources, sdktypes.NewStatementOptions{})
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txOpts := &types.TxOption{Mode: &broadcastMode}

	c, cancelPutPolicy := context.WithCancel(globalContext)
	defer cancelPutPolicy()

	statements := []*permTypes.Statement{&statement}
	policyTx, err := client.PutBucketPolicy(c, bucketName, principal, statements,
		sdktypes.PutPolicyOption{TxOpts: txOpts})

	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("put policy of the bucket:%s succ, txn hash: %s\n", bucketName, policyTx)

	if groupId > 0 {
		policyInfo, err := client.GetBucketPolicyOfGroup(c, bucketName, groupId)
		if err == nil {
			fmt.Printf("policy info of the group: \n %s\n", policyInfo.String())
		}
	} else {
		policyInfo, err := client.GetBucketPolicy(c, bucketName, grantee)
		if err == nil {
			fmt.Printf("policy info of the account:  \n %s\n", policyInfo.String())
		}
	}

	return nil
}
