package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	spClient "github.com/bnb-chain/gnfd-go-sdk/client/sp"
	"github.com/bnb-chain/gnfd-go-sdk/keys"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/urfave/cli/v2"
)

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (*spClient.SPClient, error) {
	// generate for temp test, it should fetch private key from keystore
	privKey, _, _ := testdata.KeyEthSecp256k1TestPubAddr()

	endpoint := ctx.String("endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("parse endpoint from config file fail")
	}

	if len(endpoint) <= 7 {
		return nil, fmt.Errorf("endpoint length error")
	}

	keyManager, err := keys.NewPrivateKeyManager(hex.EncodeToString(privKey.Bytes()))
	if err != nil {
		log.Fatal("new key manager fail", err.Error())
	}

	client, err := spClient.NewSpClientWithKeyManager(endpoint[7:], &spClient.Option{}, keyManager)
	if err != nil {
		log.Println("create client fail")
	}

	host := ctx.String("host")
	if host != "" {
		client.SetHost(host)
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
