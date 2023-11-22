package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

// cmdHeadObj query the object info and return
func cmdHeadObj() *cli.Command {
	return &cli.Command{
		Name:      "head",
		Action:    headObject,
		Usage:     "query object info",
		ArgsUsage: "OBJECT-URL",
		Description: `
send headObject txn to chain and fetch object info on greenfield chain
Examples:
$ gnfd-cmd object head gnfd://bucket-name/object-name`,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:    formatFlag,
				Aliases: []string{"f"},
				Value: &CmdEnumValue{
					Enum:    []string{defaultFormat, jsonFormat},
					Default: defaultFormat,
				},
				Usage: "set format of the return content of plaintxt or json",
			},
		},
	}
}

// cmdHeadBucket query the bucket info and return
func cmdHeadBucket() *cli.Command {
	return &cli.Command{
		Name:      "head",
		Action:    headBucket,
		Usage:     "query bucket info",
		ArgsUsage: "BUCKET-URL",
		Description: `
send headBucket txn to chain and fetch bucket info on greenfield chain
Examples:
$ gnfd-cmd bucket head gnfd://bucket-name`,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:    formatFlag,
				Aliases: []string{"f"},
				Value: &CmdEnumValue{
					Enum:    []string{defaultFormat, jsonFormat},
					Default: defaultFormat,
				},
				Usage: "set format of the return content of plaintxt or json",
			},
		},
	}
}

// cmdHeadGroup query the group info and return
func cmdHeadGroup() *cli.Command {
	return &cli.Command{
		Name:      "head",
		Action:    headGroup,
		Usage:     "query group info",
		ArgsUsage: "GROUP-NAME",
		Description: `
send headGroup txn to chain and fetch bucketInfo on greenfield chain
Examples:
$ gnfd-cmd group head --groupOwner  group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  groupOwnerFlag,
				Value: "",
				Usage: "need set the owner address if you are not the owner of the group",
			},
			&cli.GenericFlag{
				Name:    formatFlag,
				Aliases: []string{"f"},
				Value: &CmdEnumValue{
					Enum:    []string{defaultFormat, jsonFormat},
					Default: defaultFormat,
				},
				Usage: "set format of the return content of plaintxt or json",
			},
		},
	}
}

// cmdHeadGroupMember query the group member if it exists in group
func cmdHeadGroupMember() *cli.Command {
	return &cli.Command{
		Name:      "head-member",
		Action:    headGroupMember,
		Usage:     "check if a group member exists",
		ArgsUsage: "[Member-Address] GROUP-NAME",
		Description: `
send headGroupMember txn to chain and check if member is in the group

Examples:
$ gnfd-cmd head-member 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d  group-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  groupOwnerFlag,
				Value: "",
				Usage: "need set the owner address if you are not the owner of the group",
			},
		},
	}
}

func headObject(ctx *cli.Context) error {
	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadObject := context.WithCancel(globalContext)
	defer cancelHeadObject()

	objectDetail, err := client.HeadObject(c, bucketName, objectName)
	if err != nil {
		fmt.Println("no such object")
		return nil
	}

	fmt.Println("latest object info:")
	if format := ctx.String(formatFlag); format != "" {
		if format == defaultFormat {
			parseObjectInfo(objectDetail)
		} else if format == jsonFormat {
			parseObjectByJsonFormat(objectDetail)
		} else {
			return toCmdErr(fmt.Errorf("invalid format"))
		}
	}
	return nil
}

func headBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadBucket := context.WithCancel(globalContext)
	defer cancelHeadBucket()

	bucketInfo, err := client.HeadBucket(c, bucketName)
	if err != nil {
		fmt.Println("no such bucket")
		return nil
	}

	fmt.Println("latest bucket info:")
	if format := ctx.String(formatFlag); format != "" {
		if format == defaultFormat {
			parseBucketInfo(bucketInfo)
		} else if format == jsonFormat {
			parseBucketByJsonFormat(bucketInfo)
		} else {
			return toCmdErr(fmt.Errorf("invalid format"))
		}
	}
	return nil
}

func headGroup(ctx *cli.Context) error {
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadGroup := context.WithCancel(globalContext)
	defer cancelHeadGroup()

	groupOwner, err := getGroupOwner(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	groupInfo, err := client.HeadGroup(c, groupName, groupOwner)
	if err != nil {
		fmt.Println("no such group")
		return nil
	}

	fmt.Println("latest group info:")
	if format := ctx.String(formatFlag); format != "" {
		if format == defaultFormat {
			infoStr := strings.Split(groupInfo.String(), " ")
			for _, info := range infoStr {
				fmt.Println(info)
			}
		} else if format == jsonFormat {
			parseGroupByFormat(groupInfo)
		} else {
			return toCmdErr(fmt.Errorf("invalid format"))
		}
	}

	return nil
}

func headGroupMember(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return toCmdErr(fmt.Errorf("args number should be 2"))
	}

	// read the head member address
	headMember := ctx.Args().Get(0)
	groupName := ctx.Args().Get(1)

	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadBucket := context.WithCancel(globalContext)
	defer cancelHeadBucket()

	groupOwner, err := getGroupOwner(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	exist := client.HeadGroupMember(c, groupName, groupOwner, headMember)
	if !exist {
		fmt.Println("the user does not exist in the group")
		return nil
	}

	fmt.Printf("the user %s is a member of the group: %s \n", headMember, groupName)
	return nil
}
