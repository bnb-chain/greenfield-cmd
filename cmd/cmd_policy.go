package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	"github.com/bnb-chain/greenfield-go-sdk/pkg/utils"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	permTypes "github.com/bnb-chain/greenfield/x/permission/types"
	"github.com/urfave/cli/v2"
)

type ResourceType int

// cmdPutjPolicy set the policy of object
func cmdPutPolicy() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    putPolicy,
		Usage:     "put policy to group or account",
		ArgsUsage: " RESOURCE-URL",
		Description: `
The command is used to set the object policy of the grantee or group-id.
It required to set grantee account or group-id by --grantee or --groupId.

the resource url can be the follow types:
1) grn:b::bucketname, it indicates the bucket policy
2) grn:o::bucketname/objectname, it indicates the object policy
3) grn:g:owneraddress:groupname, it indicates the group policy

if your need to set a group policy, you need set the owneraddress as your own account address.

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
					"if it is an object policy, actions can be the following: create, delete, copy, get, execute, list, update or all," +
					"if it is a bucket policy, actions can be the following: delete, update, deleteObj, copyObj, getObj, executeObj, list or all" +
					" the actions which contain Obj means it is a action for the objects in the bucket, for example," +
					" the deleteObj means grant the permission of delete Objects in the bucket" +
					"if it is a group policy, actions can be the following: update, delete or all, update indicates the update-group-member action" +
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

func cmdDelPolicy() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Action:    deletePolicy,
		Usage:     "delete policy of principal",
		ArgsUsage: " RESOURCE-URL",
		Description: `
The command is used to set the object policy of the grantee or group-id.
It required to set grantee account or group-id by --grantee or --groupId.

the resource url can be the follow types:
1) grn:b::bucketname, it indicates the bucket policy
2) grn:o::bucketname/objectname, it indicates the object policy
3) grn:g:owneraddress:groupname, it indicates the group policy

Examples:
$ gnfd-cmd policy delete --groupId 111  grn:o::gnfd-bucket/gnfd-object`,
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
		},
	}
}

func putPolicy(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	var resourceType ResourceType
	resource := ctx.Args().Get(0)
	resourceType, err := parseResourceType(resource)
	if err != nil {
		return err
	}

	actions, err := parseActions(ctx, resourceType)
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

	return handlePutPolicy(ctx, resource, statements, resourceType)
}

func deletePolicy(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	resource := ctx.Args().Get(0)
	resourceType, err := parseResourceType(resource)
	if err != nil {
		return err
	}
	return handleDeletePolicy(ctx, resource, resourceType)
}

func handlePutPolicy(ctx *cli.Context, resource string, statements []*permTypes.Statement, policyType ResourceType) error {
	client, err := NewClient(ctx)
	if err != nil {
		return err
	}
	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)

	if policyType == BucketResourceType {
		bucketName, err := parseBucketResource(resource)
		if err != nil {
			return toCmdErr(err)
		}

		principal, err := parsePrincipal(grantee, groupId)
		if err != nil {
			return toCmdErr(err)
		}

		err = handleBucketPolicy(ctx, client, bucketName, principal, statements, false)
		if err != nil {
			return toCmdErr(err)
		}
	} else if policyType == ObjectResourceType {
		bucketName, objectName, err := parseObjectResource(resource)
		if err != nil {
			return toCmdErr(err)
		}

		principal, err := parsePrincipal(grantee, groupId)
		if err != nil {
			return toCmdErr(err)
		}

		err = handleObjectPolicy(ctx, client, bucketName, objectName, principal, statements, false)
		if err != nil {
			return toCmdErr(err)
		}
	} else if policyType == GroupResourceType {
		_, groupName, err := parseGroupResource(resource)
		if err != nil {
			return toCmdErr(err)
		}
		err = handleGroupPolicy(ctx, client, groupName, statements, false)
		if err != nil {
			return toCmdErr(err)
		}
	}
	return nil
}

func handleDeletePolicy(ctx *cli.Context, resource string, policyType ResourceType) error {
	client, err := NewClient(ctx)
	if err != nil {
		return err
	}
	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)

	if policyType == BucketResourceType {
		bucketName, err := parseBucketResource(resource)
		if err != nil {
			return toCmdErr(err)
		}

		principal, err := parsePrincipal(grantee, groupId)
		if err != nil {
			return toCmdErr(err)
		}

		err = handleBucketPolicy(ctx, client, bucketName, principal, nil, true)
		if err != nil {
			return toCmdErr(err)
		}

	} else if policyType == ObjectResourceType {
		bucketName, objectName, err := parseObjectResource(resource)
		if err != nil {
			return toCmdErr(err)
		}

		principal, err := parsePrincipal(grantee, groupId)
		if err != nil {
			return toCmdErr(err)
		}

		err = handleObjectPolicy(ctx, client, bucketName, objectName, principal, nil, true)
		if err != nil {
			return toCmdErr(err)
		}
	} else if policyType == GroupResourceType {
		_, groupName, err := parseGroupResource(resource)
		if err != nil {
			return toCmdErr(err)
		}
		err = handleGroupPolicy(ctx, client, groupName, nil, true)
		if err != nil {
			return toCmdErr(err)
		}
	}

	return nil
}

func handleObjectPolicy(ctx *cli.Context, client client.Client, bucketName, objectName string, principal sdktypes.Principal,
	statements []*permTypes.Statement, delete bool) error {
	c, cancelObjectPolicy := context.WithCancel(globalContext)
	defer cancelObjectPolicy()

	var policyTx string
	var err error
	if !delete {
		policyTx, err = client.PutObjectPolicy(c, bucketName, objectName, principal, statements,
			sdktypes.PutPolicyOption{TxOpts: &types.TxOption{Mode: &SyncBroadcastMode}})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("put policy of the object:%s succ, txn hash: %s\n", objectName, policyTx)
	} else {
		grantee := ctx.String(granteeFlag)
		if grantee == "" {
			return errors.New("grantee need to be set when delete object policy")
		}
		policyTx, err = client.DeleteObjectPolicy(c, bucketName, objectName, grantee,
			sdktypes.DeletePolicyOption{TxOpts: &types.TxOption{Mode: &SyncBroadcastMode}})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("delete policy of the object:%s succ, txn hash: %s\n", objectName, policyTx)
	}

	err = waitTxnStatus(client, c, policyTx, "objectPolicy")
	if err != nil {
		return toCmdErr(err)
	}

	// print object policy info after updated
	printObjectPolicy(ctx, client, bucketName, objectName)

	return nil
}

func handleObjectPolicy(ctx *cli.Context, client client.Client, bucketName, objectName string, principal sdktypes.Principal,
	statements []*permTypes.Statement, delete bool) error {
	c, cancelObjectPolicy := context.WithCancel(globalContext)
	defer cancelObjectPolicy()

	var policyTx string
	var err error
	if !delete {
		policyTx, err = client.PutObjectPolicy(c, bucketName, objectName, principal, statements,
			sdktypes.PutPolicyOption{TxOpts: &types.TxOption{Mode: &SyncBroadcastMode}})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("put policy of the object:%s succ, txn hash: %s\n", objectName, policyTx)
	} else {
		policyTx, err = client.DeleteObjectPolicy(c, bucketName, objectName, principal,
			sdktypes.DeletePolicyOption{TxOpts: &types.TxOption{Mode: &SyncBroadcastMode}})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("delete policy of the object:%s succ, txn hash: %s\n", objectName, policyTx)
	}

	err = waitTxnStatus(client, c, policyTx, "objectPolicy")
	if err != nil {
		return toCmdErr(err)
	}

	// print object policy info after updated
	printObjectPolicy(ctx, client, bucketName, objectName)

	return nil
}

func handleBucketPolicy(ctx *cli.Context, client client.Client, bucketName string, principal sdktypes.Principal,
	statements []*permTypes.Statement, delete bool) error {
	c, cancelBucketPolicy := context.WithCancel(globalContext)
	defer cancelBucketPolicy()

	var policyTx string
	var err error
	if !delete {
		policyTx, err = client.PutBucketPolicy(c, bucketName, principal, statements,
			sdktypes.PutPolicyOption{TxOpts: &TxnOptionWithSyncMode})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("put policy of the bucket:%s succ, txn hash: %s\n", bucketName, policyTx)

	} else {
		policyTx, err = client.DeleteBucketPolicy(c, bucketName, principal, sdktypes.DeletePolicyOption{TxOpts: &TxnOptionWithSyncMode})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("delete policy of the bucket:%s succ, txn hash: %s\n", bucketName, policyTx)
	}

	err = waitTxnStatus(client, c, policyTx, "bucketPolicy")
	if err != nil {
		return toCmdErr(err)
	}

	// print bucket policy info after updated
	printBucketPolicy(ctx, client, bucketName)

	return nil
}

func handleGroupPolicy(ctx *cli.Context, client client.Client, groupName string,
	statements []*permTypes.Statement, delete bool) error {
	c, cancelPolicy := context.WithCancel(globalContext)
	defer cancelPolicy()

	grantee := ctx.String(granteeFlag)
	if grantee == "" {
		return errors.New("grantee need to be set when put group policy")
	}
	var policyTx string
	var err error
	if !delete {
		policyTx, err = client.PutGroupPolicy(c, groupName, grantee, statements,
			sdktypes.PutPolicyOption{TxOpts: &TxnOptionWithSyncMode})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("put policy of the group:%s succ, txn hash: %s\n", groupName, policyTx)
	} else {
		policyTx, err := client.DeleteGroupPolicy(c, groupName, grantee, sdktypes.DeletePolicyOption{TxOpts: &TxnOptionWithSyncMode})
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("delete policy of the group:%s succ, txn hash: %s\n", groupName, policyTx)
	}

	err = waitTxnStatus(client, c, policyTx, "groupPolicy")
	if err != nil {
		return toCmdErr(err)
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

func parseGroupResource(resourceName string) (string, string, error) {
	prefixLen := len(GroupResourcePrefix)

	if len(resourceName) <= prefixLen {
		return "", "", errors.New("invalid group resource name")
	}

	objectPath := resourceName[prefixLen:]
	index := strings.Index(objectPath, ":")

	if index <= -1 {
		return "", "", errors.New("invalid group resource name, can not parse bucket name and object name")
	}

	return objectPath[:index], objectPath[index+1:], nil
}

func printObjectPolicy(ctx *cli.Context, cli client.Client, bucketName, objectName string) {
	// get the latest policy from chain
	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)
	c, cancelPolicy := context.WithCancel(globalContext)
	defer cancelPolicy()
	if groupId > 0 {
		policyInfo, err := cli.GetObjectPolicyOfGroup(c, bucketName, objectName, groupId)
		if err == nil {
			fmt.Printf("latest object policy info: \n %s\n", policyInfo.String())
		}
	} else {
		policyInfo, err := cli.GetObjectPolicy(c, bucketName, objectName, grantee)
		if err == nil {
			fmt.Printf("latest object policy info:  \n %s\n", policyInfo.String())
		}
	}
}

func printBucketPolicy(ctx *cli.Context, cli client.Client, bucketName string) {
	c, cancelPolicy := context.WithCancel(globalContext)
	defer cancelPolicy()
	// get the latest policy from chain
	groupId := ctx.Uint64(groupIDFlag)
	grantee := ctx.String(granteeFlag)
	if groupId > 0 {
		policyInfo, err := cli.GetBucketPolicyOfGroup(c, bucketName, groupId)
		if err == nil {
			fmt.Printf("latest bucket policy info: \n %s\n", policyInfo.String())
		}
	} else {
		policyInfo, err := cli.GetBucketPolicy(c, bucketName, grantee)
		if err == nil {
			fmt.Printf("latest bucket policy info:  \n %s\n", policyInfo.String())
		}
	}
}

func parseResourceType(resource string) (ResourceType, error) {
	var resourceType ResourceType
	if strings.HasPrefix(resource, BucketResourcePrefix) {
		resourceType = BucketResourceType
	} else if strings.HasPrefix(resource, ObjectResourcePrefix) {
		resourceType = ObjectResourceType
	} else if strings.HasPrefix(resource, GroupResourcePrefix) {
		resourceType = GroupResourceType
	} else {
		return -1, toCmdErr(errors.New("invalid resource name"))
	}
	return resourceType, nil
}
