package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	spClient "github.com/bnb-chain/greenfield-go-sdk/client/sp"
	"github.com/urfave/cli/v2"
)

func cmdChallenge() *cli.Command {
	return &cli.Command{
		Name:      "challenge",
		Action:    getChallengeInfo,
		Usage:     "send challenge request",
		ArgsUsage: "",
		Description: `
Send challenge request to the storage provider. 
The command need to need to set the object id, piece index and sp index.

Examples:
$ gnfd-cmd -c config.toml challenge --objectId "test" --pieceIndex 2  --spIndex -1`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     objectIDFlagName,
				Value:    "",
				Usage:    "the objectId to be challenge",
				Required: true,
			},
			&cli.IntFlag{
				Name:     pieceIndexFlagName,
				Value:    0,
				Usage:    "show which piece to be challenge",
				Required: true,
			},
			&cli.IntFlag{
				Name:     spIndexFlagName,
				Value:    -1,
				Usage:    "indicate the challenge sp index",
				Required: true,
			},
		},
	}
}

func getChallengeInfo(ctx *cli.Context) error {
	objectId := ctx.String(objectIDFlagName)
	if objectId == "" {
		return toCmdErr(errors.New("object id empty "))
	}

	pieceIndex := ctx.Int(pieceIndexFlagName)
	if pieceIndex < 0 {
		return toCmdErr(errors.New("pieceIndex should not be less than 0 "))
	}

	spIndex := ctx.Int(spIndexFlagName)
	if spIndex < -1 {
		return toCmdErr(errors.New("redundancyIndex should not be less than -1"))
	}

	s3Client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	filePath := ctx.Args().Get(0)

	st, err := os.Stat(filePath)
	if err == nil {
		// If the destination exists and is a directory.
		if st.IsDir() {
			return toCmdErr(errors.New("fileName is a directory"))
		}
	}

	// If file exist, open it in append mode
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return toCmdErr(err)
	}

	info := spClient.ChallengeInfo{
		ObjectId:        objectId,
		PieceIndex:      pieceIndex,
		RedundancyIndex: spIndex,
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	res, err := s3Client.SPClient.ChallengeSP(c, info, spClient.NewAuthInfo(false, ""))
	if err != nil {
		fmt.Println("fail to challenge:", err.Error())
		return nil
	}

	if res.PiecesHash != nil {
		fmt.Println("get hash result", res.PiecesHash)
	} else {
		return toCmdErr(errors.New("fail to fetch piece hashes"))
	}

	if res.PieceData != nil {
		defer res.PieceData.Close()
		_, err = io.Copy(fd, res.PieceData)
		fd.Close()
		if err != nil {
			return toCmdErr(err)
		}

		fmt.Printf("download challenge payload into file:%s successfully \n", filePath)
	} else {
		return toCmdErr(errors.New("fail to fetch challenge data"))
	}

	return nil
}
