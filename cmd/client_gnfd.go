package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sdkTypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/urfave/cli/v2"
)

const iso8601DateFormatSecond = "2006-01-02T15:04:05Z"

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (client.Client, error) {
	grpcAddr := ctx.String("rpcAddr")
	if grpcAddr == "" {
		return nil, fmt.Errorf("failed to parse grpc address, please set it in the config file")
	}

	chainId := ctx.String("chainId")
	if chainId == "" {
		return nil, fmt.Errorf("failed to parse chain id, please set it in the config file")
	}

	privateKeyStr := ctx.String("privateKey")
	if privateKeyStr == "" {
		return nil, fmt.Errorf("failed to private key")
	}

	account, err := sdkTypes.NewAccountFromPrivateKey("gnfd-account", privateKeyStr)
	if err != nil {
		return nil, err
	}

	var cli client.Client
	host := ctx.String("host")
	if host != "" {
		cli, err = client.New(chainId, grpcAddr, client.Option{DefaultAccount: account, Host: host})
	} else {
		cli, err = client.New(chainId, grpcAddr, client.Option{DefaultAccount: account})
	}

	if err != nil {
		fmt.Println("failed to create client" + err.Error())
	}

	return cli, err
}

// ParseBucketAndObject parse the bucket-name and object-name from url
func ParseBucketAndObject(urlPath string) (string, string, error) {
	if strings.Contains(urlPath, "gnfd://") {
		urlPath = urlPath[len("gnfd://"):]
	}

	index := strings.Index(urlPath, "/")

	if index <= -1 {
		return "", "", errors.New("url not right, can not parse bucket name and object name")
	}

	return urlPath[:index], urlPath[index+1:], nil
}

// ParseBucket parse the bucket-name from url
func ParseBucket(urlPath string) (bucketName string) {
	if strings.Contains(urlPath, "gnfd://") {
		urlPath = urlPath[len("gnfd://"):]
	}
	splits := strings.SplitN(urlPath, "/", 1)

	return splits[0]
}
