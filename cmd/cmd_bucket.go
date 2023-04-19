package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	sdkTypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdCreateBucket create a new Bucket
func cmdCreateBucket() *cli.Command {
	return &cli.Command{
		Name:      "make-bucket",
		Action:    createBucket,
		Usage:     "create a new bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Create a new bucket and set a createBucketMsg to storage provider.
The bucket name should unique and the default visibility is private.
The command need to set the primary SP address with --primarySP.

Examples:
# Create a new bucket called gnfdBucket, visibility is public-read
$ gnfd-cmd -c config.toml make-bucket --primarySP  --visibility=public-read  gnfd://gnfdBucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  primarySPFlagName,
				Value: "",
				Usage: "indicate the primarySP address, using the string type",
			},
			&cli.StringFlag{
				Name:  paymentFlagName,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlagName,
				Value: 0,
				Usage: "indicate the read quota info of the bucket",
			},
			&cli.GenericFlag{
				Name: visibilityFlagName,
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
		Name:      "update-bucket",
		Action:    updateBucket,
		Usage:     "update bucket meta on chain",
		ArgsUsage: "BUCKET-URL",
		Description: `
Update the visibility, payment account or read quota meta of the bucket.
The visibility value can be public-read, private or inherit.
You can update only one item or multiple items at the same time.

Examples:
update visibility and the payment address of the gnfdBucket
$ gnfd-cmd -c config.toml update-bucket --visibility=public-read --paymentAddress xx  gnfd://gnfdBucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  paymentFlagName,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlagName,
				Usage: "indicate the read quota info of the bucket",
			},
			&cli.GenericFlag{
				Name: visibilityFlagName,
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
		Name:      "ls-bucket",
		Action:    listBuckets,
		Usage:     "list buckets of the user",
		ArgsUsage: "",
		Description: `
List the bucket names and bucket ids of the user.

Examples:
$ gnfd-cmd -c config.toml ls-bucket `,
	}
}

func cmdPutBucketPolicy() *cli.Command {
	return &cli.Command{
		Name:      "put-bucket-policy",
		Action:    putBucketPolicy,
		Usage:     "put bucket policy to group or account",
		ArgsUsage: " BUCKET-URL",
		Description: `
The command is used to set the bucket policy of the granted account or group-id.
It required to set granted account or group-id by --groupId or --granter.

Examples:
$ gnfd-cmd -c config.toml put-bucket-policy --groupId 111 --action delete,update gnfd://gnfdBucket/gnfdObject`,
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
					"actions can be the following: delete, update." +
					" multi actions like \"delete,update\" is supported",
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

	primarySpAddrStr := ctx.String(primarySPFlagName)
	if primarySpAddrStr == "" {
		return toCmdErr(errors.New("primary sp address must be set"))
	}

	opts := sdkTypes.CreateBucketOptions{}
	paymentAddrStr := ctx.String(paymentFlagName)
	if paymentAddrStr != "" {
		opts.PaymentAddress = sdk.MustAccAddressFromHex(paymentAddrStr)
	}

	visibility := ctx.Generic(visibilityFlagName)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlagName)
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

	opts := sdkTypes.UpdateBucketOption{}
	paymentAddrStr := ctx.String(paymentFlagName)
	if paymentAddrStr != "" {
		paymentAddress, err := sdk.AccAddressFromHexUnsafe(paymentAddrStr)
		if err != nil {
			return err
		}
		opts.PaymentAddress = paymentAddress
	}

	visibility := ctx.Generic(visibilityFlagName)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlagName)
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
		fmt.Printf("bucket name: %s, bucket id: %s \n", info.BucketName, info.Id)
	}
	return nil

}

func putBucketPolicy(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	groupId := ctx.Uint64(groupIDFlagName)
	granter := ctx.String(granterFlagName)
	principal, err := parsePrincipal(ctx, granter, groupId)
	if err != nil {
		return toCmdErr(err)
	}

	actions, err := parseActions(ctx, false)
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
	policyTx, err := client.PutBucketPolicy(bucketName, principal, statements,
		gnfdclient.PutPolicyOption{TxOpts: txOpts})

	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("put bucket policy %s succ, txn hash: %s\n", bucketName, policyTx)

	c, cancelPutPolicy := context.WithCancel(globalContext)
	defer cancelPutPolicy()

	if groupId > 0 {
		policyInfo, err := client.GetBucketPolicyOfGroup(c, bucketName, groupId)
		if err == nil {
			fmt.Printf("policy info of the group: \n %s\n", policyInfo.String())
		}
	} else {
		granterAddr, err := sdk.AccAddressFromHexUnsafe(granter)
		if err == nil {
			policyInfo, err := client.GetBucketPolicy(c, bucketName, granterAddr)
			if err == nil {
				fmt.Printf("policy info of the account:  \n %s\n", policyInfo.String())
			}
		}
	}

	return nil
}
