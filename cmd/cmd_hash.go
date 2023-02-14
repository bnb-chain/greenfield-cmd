package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

// cmdMakeBucket create a new Bucket
func cmdCalHash() *cli.Command {
	return &cli.Command{
		Name:      "get-hash",
		Action:    computeHashRoot,
		Usage:     "compute hash roots of object ",
		ArgsUsage: "filePath",
		Description: `

Examples:
# Compute file path
$ gnfd get-hash --segSize 16  --shards 6 /home/test.text `,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "segSize",
				Value:    16,
				Usage:    "the segment size (MB)",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "shards",
				Value:    6,
				Usage:    "the ec encode shard number",
				Required: true,
			},
		},
	}
}

func computeHashRoot(ctx *cli.Context) error {
	// read the local file payload to be uploaded
	filePath := ctx.Args().Get(0)

	exists, objectSize, err := pathExists(filePath)
	if !exists {
		return errors.New("upload file not exists")
	} else if objectSize > int64(500*1024*1024) {
		return errors.New("upload file larger than 500M ")
	}

	// Open the referenced file.
	fReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fReader.Close()

	segmentSize := ctx.Int("segSize")
	if segmentSize <= 0 {
		return errors.New("segment size should be more than 0 ")
	}

	ecShards := ctx.Int("shards")
	if ecShards <= 0 {
		return errors.New("encode shards number should be more than 0 ")
	}

	s3Client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	priHash, secondHash, _, err := s3Client.GetPieceHashRoots(fReader, int64(segmentSize*1024*1024), ecShards)
	if err != nil {
		return err
	}

	fmt.Printf("get primary sp hash root: \n%s\n", priHash)
	fmt.Println("get secondary sp hash list:")
	for _, hash := range secondHash {
		fmt.Println(hash)
	}

	return nil
}
