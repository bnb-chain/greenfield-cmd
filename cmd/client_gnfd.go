package main

import (
	"context"
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
	configFile := ctx.String("config")
	var config *cmdConfig
	var err error
	if configFile != "" {
		config, err = parseConfigFile(configFile)
		if err != nil {
			return nil, err
		}
	}

	rpcAddr := ctx.String("rpcAddr")
	if rpcAddr == "" {
		if config.RpcAddr == "" {
			return nil, fmt.Errorf("failed to parse rpc address, please set it in the config file")
		} else {
			rpcAddr = config.RpcAddr
		}
	}

	chainId := ctx.String("chainId")
	if chainId == "" {
		if config.ChainId == "" {
			return nil, fmt.Errorf("failed to parse chain id, please set it in the config file")
		} else {
			chainId = config.ChainId
		}
	}

	keyfilepath := ctx.String("keystore")
	if keyfilepath == "" {
		keyfilepath = defaultKeyfile
	}

	// fetch private key from keystore
	keyjson, err := os.ReadFile(keyfilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the keyfile at '%s': %v \n", keyfilepath, err)
	}

	password, err := getPassword(ctx, config)
	if err != nil {
		return nil, err
	}

	privateKey, err := DecryptKey(keyjson, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypting key: %v \n", err)
	}

	account, err := sdktypes.NewAccountFromPrivateKey("gnfd-account", privateKey)
	if err != nil {
		fmt.Println("new account err", err.Error())
		return nil, err
	}

	var cli client.Client
	host := ctx.String("host")
	if host == "" && config.Host != "" {
		host = config.Host
	}

	if host != "" {
		cli, err = client.New(chainId, rpcAddr, client.Option{DefaultAccount: account, Host: host})
	} else {
		cli, err = client.New(chainId, rpcAddr, client.Option{DefaultAccount: account})
	}

	if err != nil {
		fmt.Printf("failed to create client %s \n", err.Error())
		return nil, err
	}

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

func waitTxnStatus(cli client.Client, ctx context.Context, txnHash string, txnInfo string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, ContextTimeout)
	defer cancel()

	txnResponse, err := cli.WaitForTx(ctxTimeout, txnHash)
	if err != nil {
		return fmt.Errorf("the %s txn: %s ,has been submitted, please check it later:%v", txnInfo, txnHash, err)
	}
	if txnResponse.Code != 0 {
		return fmt.Errorf("the %s txn: %s has failed with response code: %d", txnInfo, txnHash, txnResponse.Code)
	}

	return nil
}
