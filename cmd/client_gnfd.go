package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/urfave/cli/v2"
)

const iso8601DateFormatSecond = "2006-01-02T15:04:05Z"

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (client.Client, error) {
	rpcAddr := ctx.String("rpcAddr")
	if rpcAddr == "" {
		return nil, fmt.Errorf("failed to parse rpc address, please set it in the config file")
	}

	chainId := ctx.String("chainId")
	if chainId == "" {
		return nil, fmt.Errorf("failed to parse chain id, please set it in the config file")
	}

	keyfilepath := ctx.String("keystore")
	if keyfilepath == "" {
		keyfilepath = defaultKeyfile
	}

	// fetch private key from keystore
	keyjson, err := os.ReadFile(keyfilepath)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read the keyfile at '%s': %v \n", keyfilepath, err))
	}

	password, err := getPassword(ctx)
	if err != nil {
		return nil, err
	}

	privateKey, err := DecryptKey(keyjson, password)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to decrypting key: %v \n", err))
	}

	fmt.Println("decrypt key:", privateKey)

	account, err := sdktypes.NewAccountFromPrivateKey("gnfd-account", privateKey)
	if err != nil {
		fmt.Println("new account err", err.Error())
		return nil, err
	}

	var cli client.Client
	host := ctx.String("host")
	if host != "" {
		cli, err = client.New(chainId, rpcAddr, client.Option{DefaultAccount: account, Host: host})
	} else {
		cli, err = client.New(chainId, rpcAddr, client.Option{DefaultAccount: account})
	}

	if err != nil {
		fmt.Printf("failed to create client %s \n", err.Error())
		return nil, err
	}

	cli.EnableTrace(nil, false)
	return cli, nil
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
