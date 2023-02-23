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

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (*gnfdclient.GnfdClient, error) {
	// generate for temp test, it should fetch private key from keystore
	privKey, _, _ := testdata.KeyEthSecp256k1TestPubAddr()

	endpoint := ctx.String("endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("parse endpoint from config file fail")
	}

	grpcAddr := ctx.String("grpcAddr")
	if grpcAddr == "" {
		return nil, fmt.Errorf("parse grpc address from config file fail")
	}

	chainId := ctx.String("chainId")
	if chainId == "" {
		return nil, fmt.Errorf("parse chain id from config file fail")
	}

	keyManager, err := keys.NewPrivateKeyManager(hex.EncodeToString(privKey.Bytes()))
	if err != nil {
		log.Error().Msg("new key manager fail" + err.Error())
	}

	client, err := gnfdclient.NewGnfdClient(grpcAddr, chainId, endpoint, keyManager, false,
		WithGrpcDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))

	if err != nil {
		fmt.Println("create client fail" + err.Error())
	}

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
