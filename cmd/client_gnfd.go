package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	"github.com/bnb-chain/greenfield/sdk/client"
	"github.com/bnb-chain/greenfield/sdk/keys"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var NewPrivateKeyManager = keys.NewPrivateKeyManager
var WithGrpcDialOption = client.WithGrpcDialOption
var WithKeyManager = client.WithKeyManager

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (*gnfdclient.GnfdClient, error) {
	endpoint := ctx.String("endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("parse endpoint from config file fail")
	}

	if strings.Contains(endpoint, "http") {
		endpoint = endpoint[7:]
	}

	grpcAddr := ctx.String("grpcAddr")
	if grpcAddr == "" {
		return nil, fmt.Errorf("parse grpc address from config file fail")
	}

	chainId := ctx.String("chainId")
	if chainId == "" {
		return nil, fmt.Errorf("parse chain id from config file fail")
	}

	privateKeyStr := ctx.String("privateKey")
	if privateKeyStr == "" {
		// generate private key if not provided
		privKey, _, _ := testdata.KeyEthSecp256k1TestPubAddr()
		privateKeyStr = hex.EncodeToString(privKey.Bytes())
	}

	keyManager, err := keys.NewPrivateKeyManager(privateKeyStr)
	if err != nil {
		log.Error().Msg("new key manager fail" + err.Error())
	}

	client, err := gnfdclient.NewGnfdClient(grpcAddr, chainId, endpoint, keyManager, false,
		WithKeyManager(keyManager),
		WithGrpcDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		fmt.Println("create client fail" + err.Error())
	}

	fmt.Println("sender addr is:", client.SPClient.GetAccount().String(), " ,the address should have balance to test")

	host := ctx.String("host")
	if host != "" {
		client.SPClient.SetHost(host)
	}

	return client, err
}

// ParseBucketAndObject parse the bucket-name and object-name from url
func ParseBucketAndObject(urlPath string) (bucketName, objectName string) {
	if strings.Contains(urlPath, "gnfd://") {
		urlPath = urlPath[len("gnfd://"):]
	}
	splits := strings.SplitN(urlPath, "/", 2)
	return splits[0], splits[1]
}

// ParseBucket parse the bucket-name from url
func ParseBucket(urlPath string) (bucketName string) {
	if strings.Contains(urlPath, "gnfd://") {
		urlPath = urlPath[len("gnfd://"):]
	}
	splits := strings.SplitN(urlPath, "/", 1)

	return splits[0]
}
