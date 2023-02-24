package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// cmdGetObj return the command to finish downloading object payload
func cmdGetObj() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Action:    getObject,
		Usage:     "download object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Download a specific object from storage provider

Examples:
# download a file
$ gnfd get gnfd://bucketname/file.txt file.txt `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "filepath",
				Value: "",
				Usage: "file path info to be uploaded",
			},
		},
	}
}

// getObject download the object payload from sp
func getObject(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("the args number should be two")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName := ParseBucketAndObject(urlInfo)

	gnfdClient, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	// filePath := ctx.String("filepath")
	filePath := ctx.Args().Get(1)
	log.Printf("download object %s into file:%s \n", objectName, filePath)

	// If file exist, open it in append mode
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	defer fd.Close()

	body, _, err := gnfdClient.DownloadObject(c, bucketName, objectName)
	if err != nil {
		return err
	}

	_, err = io.Copy(fd, body)
	if err != nil {
		return err
	}

	log.Println("downaload object finish")
	return nil
}
