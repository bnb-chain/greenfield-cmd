package main

import (
	"context"
	"fmt"
	"log"

	"github.com/urfave/cli/v2"

	inscription "github.com/bnb-chain/greenfield-sdk-go"
)

// cmdGetObj return the command to finish downloading object payload
func cmdGetObj() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Action:    getObject,
		Usage:     "Download object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Download a specific object from storage provider

Examples:
# download a file
$ gnfd get s3://bucketname/file.txt file.txt `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "filepath",
				Value: "",
				Usage: "file path info to be uploaded",
			},
		},
	}
}

func getObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("the args number should be two")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName := ParseBucketAndObject(urlInfo)

	s3Client, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	// filePath := ctx.String("filepath")
	filePath := ctx.Args().Get(1)
	log.Printf("download object %s into file:%s \n", objectName, filePath)

	err = s3Client.FGetObject(c, bucketName, objectName, filePath, inscription.GetObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}
