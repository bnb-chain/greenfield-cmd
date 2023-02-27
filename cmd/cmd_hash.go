package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
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
$ gnfd-cmd get-hash --segSize 16  --dataShards 4 --parityShards 2 /home/test.text `,
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:  "segSize",
				Value: 16,
				Usage: "the segment size (MB)",
			},
			&cli.Uint64Flag{
				Name:  "dataShards",
				Value: 4,
				Usage: "the ec encode shard number",
			},
			&cli.Uint64Flag{
				Name:  "parityShards",
				Value: 2,
				Usage: "the ec encode shard number",
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

	opts := gnfdclient.ComputeHashOptions{}
	segmentSize := ctx.Uint64("segSize")
	if segmentSize > 0 {
		opts.SegmentSize = segmentSize
	}

	dataBlocks := ctx.Uint64("dataShards")
	if dataBlocks > 0 {
		opts.DataShards = uint32(dataBlocks)
	}

	parityBlocks := ctx.Uint64("parityShards")
	if parityBlocks > 0 {
		opts.ParityShards = uint32(parityBlocks)
	}

	gnfdClient, err := NewClient(ctx)
	if err != nil {
		return err
	}

	// Open the referenced file.
	fReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fReader.Close()

	hashes, size, err := gnfdClient.ComputeHash(fReader, opts)
	if err != nil {
		return err
	}

	fmt.Printf("get primary sp hash root: \n%s\n", hashes[0])
	fmt.Println("get secondary sp hash list:")
	for _, hash := range hashes[1:] {
		fmt.Println(hash)
	}
	fmt.Println("file size:", size)

	return nil
}
