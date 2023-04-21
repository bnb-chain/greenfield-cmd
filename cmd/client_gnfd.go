package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	"github.com/bnb-chain/greenfield/sdk/client"
	"github.com/bnb-chain/greenfield/sdk/keys"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var NewPrivateKeyManager = keys.NewPrivateKeyManager
var WithGrpcDialOption = client.WithGrpcDialOption
var WithKeyManager = client.WithKeyManager

const iso8601DateFormatSecond = "2006-01-02T15:04:05Z"

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (*gnfdclient.GnfdClient, error) {
	endpoint := ctx.String("endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("failed to parse endpoint, please set it in the config file")
	}

	if strings.Contains(endpoint, "http") {
		s := strings.Split(endpoint, "//")
		endpoint = s[1]
	}

	grpcAddr := ctx.String("grpcAddr")
	if grpcAddr == "" {
		return nil, fmt.Errorf("failed to parse grpc address, please set it in the config file")
	}

	chainId := ctx.String("chainId")
	if chainId == "" {
		return nil, fmt.Errorf("failed to parse chain id, please set it in the config file")
	}

	privateKeyStr := ctx.String("privateKey")
	if privateKeyStr == "" {
		// generate private key if not provided
		privKey, _, _ := testdata.KeyEthSecp256k1TestPubAddr()
		privateKeyStr = hex.EncodeToString(privKey.Bytes())
	}

	keyManager, err := keys.NewPrivateKeyManager(privateKeyStr)
	if err != nil {
		return nil, err
	}

	client, err := gnfdclient.NewGnfdClient(grpcAddr, chainId, endpoint, keyManager, false,
		WithKeyManager(keyManager),
		WithGrpcDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		return nil, err
	}

	host := ctx.String("host")
	if host != "" {
		client.SPClient.SetHost(host)
	}

	return client, nil
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
