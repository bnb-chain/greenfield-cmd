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
	"github.com/urfave/cli/v2"
)

type policyType int

// cmdPutjPolicy set the policy of object
func cmdPutPolicy() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    putPolicy,
		Usage:     "put object policy to group or account",
		ArgsUsage: " RESOURCE-URL",
		Description: `
The command is used to set the object policy of the grantee or group-id.
It required to set grantee account or group-id by --grantee or --groupId.

the resource url can be the follow pattern:
1) grn:b::bucketname, it indicates the bucket policy
2) grn:o::bucketname/objectname, it indicates the object policy
3) grn:g:owneraddress:groupname, it indicates the group policy

Examples:
$ gnfd-cmd policy put --groupId 111 --actions get,delete grn:o::gnfd-bucket/gnfd-object`,
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
					"if it is an object policy, actions can be the following: create, delete, copy, get, execute, list or all," +
					"if it is a bucket policy, actions can be the following: delete, update, deleteObj, copyObj, getObj, executeObj, list or all" +
					" the actions which contain Obj means it is a action for the objects in the bucket, for example," +
					" the deleteObj means grant the permission of delete Objects in the bucket" +
					", multi actions like \"delete,copy\" is supported",
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

func putPolicy(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	var putPolicyType policyType
	resource := ctx.Args().Get(0)
	if strings.HasPrefix(resource, BucketResourcePrefix) {
		putPolicyType = BucketPolicyType
	} else if strings.HasPrefix(resource, ObjectResourcePrefix) {
		putPolicyType = ObjectPolicyType
	} else if strings.HasPrefix(resource, GroupResourcePrefix) {
		putPolicyType = GroupPolicyType
	} else {
		return toCmdErr(errors.New("invalid resour name"))
	}

	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)
	principal, err := parsePrincipal(grantee, groupId)
	if err != nil {
		return toCmdErr(err)
	}

	actions, err := parseActions(ctx, putPolicyType)
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

	expireTime := ctx.Uint64(expireTimeFlag)
	var statement permTypes.Statement
	if expireTime > 0 {
		tm := time.Unix(int64(expireTime), 0)
		statement = utils.NewStatement(actions, effect, nil, sdktypes.NewStatementOptions{StatementExpireTime: &tm})
	} else {
		statement = utils.NewStatement(actions, effect, nil, sdktypes.NewStatementOptions{})
	}

	statements := []*permTypes.Statement{&statement}

	if strings.HasPrefix(resource, BucketResourcePrefix) {
		bucketName, err := parseBucketResource(resource)
		if err != nil {
			return toCmdErr(err)
		}
		err = handleBucketPolicy(ctx, bucketName, principal, statements)
		if err != nil {
			return toCmdErr(err)
		}
	} else if strings.HasPrefix(resource, ObjectResourcePrefix) {
		bucketName, objectName, err := parseObjectResource(resource)
		if err != nil {
			return toCmdErr(err)
		}
		err = handleObjectPolicy(ctx, bucketName, objectName, principal, statements)
		if err != nil {
			return toCmdErr(err)
		}
	}

	return nil
}

func handleObjectPolicy(ctx *cli.Context, bucketName, objectName string, principal sdktypes.Principal,
	statements []*permTypes.Statement) error {
	client, err := NewClient(ctx)
	if err != nil {
		return err
	}
	c, cancelPutPolicy := context.WithCancel(globalContext)
	defer cancelPutPolicy()
	policyTx, err := client.PutObjectPolicy(c, bucketName, objectName, principal, statements,
		sdktypes.PutPolicyOption{TxOpts: &types.TxOption{Mode: &SyncBroadcastMode}})

	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("put policy of the object:%s succ, txn hash: %s\n", objectName, policyTx)

	_, err = client.WaitForTx(c, policyTx)
	if err != nil {
		return toCmdErr(errors.New("failed to commit put policy txn:" + err.Error()))
	}
	// get the latest policy from chain
	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)
	if groupId > 0 {
		policyInfo, err := client.GetObjectPolicyOfGroup(c, bucketName, objectName, groupId)
		if err == nil {
			fmt.Printf("policy info of the group: \n %s\n", policyInfo.String())
		}
	} else {
		policyInfo, err := client.GetObjectPolicy(c, bucketName, objectName, grantee)
		if err == nil {
			fmt.Printf("policy info of the account:  \n %s\n", policyInfo.String())
		}
	}

	return nil
}

func handleBucketPolicy(ctx *cli.Context, bucketName string, principal sdktypes.Principal,
	statements []*permTypes.Statement) error {
	c, cancelPutPolicy := context.WithCancel(globalContext)
	defer cancelPutPolicy()

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	policyTx, err := client.PutBucketPolicy(c, bucketName, principal, statements,
		sdktypes.PutPolicyOption{TxOpts: &TxnOptionWithSyncMode})

	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("put policy of the bucket:%s succ, txn hash: %s\n", bucketName, policyTx)

	_, err = client.WaitForTx(c, policyTx)
	if err != nil {
		return toCmdErr(errors.New("failed to commit put policy txn:" + err.Error()))
	}

	// get the latest policy from chain
	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)
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

func parseBucketResource(resourceName string) (string, error) {
	prefixLen := len(BucketResourcePrefix)
	if len(resourceName) <= prefixLen {
		return "", errors.New("invalid bucket resource name")
	}

	return resourceName[prefixLen:], nil
}

func parseObjectResource(resourceName string) (string, string, error) {
	prefixLen := len(ObjectResourcePrefix)

	if len(resourceName) <= prefixLen {
		return "", "", errors.New("invalid object resource name")
	}

	objectPath := resourceName[prefixLen:]
	index := strings.Index(objectPath, "/")

	if index <= -1 {
		return "", "", errors.New("invalid object resource name, can not parse bucket name and object name")
	}

	return objectPath[:index], objectPath[index+1:], nil
}
