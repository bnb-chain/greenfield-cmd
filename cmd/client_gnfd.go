package main

import (
	"fmt"
	"log"
	"strings"

	inscription "github.com/bnb-chain/greenfield-sdk-go"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/urfave/cli/v2"
)

// GnfdClient construct
type GnfdClient struct {
	args  []string
	clint inscription.Client
}

func NewClient(ctx *cli.Context) (*inscription.Client, error) {
	// generate for temp test, it should fetch private key from keystore
	privKey, pubKey, addr := testdata.KeyEthSecp256k1TestPubAddr()

	endpoint := ctx.String("endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("parse endpoint from config file fail")
	}

	fmt.Println("parse config endpoint:", endpoint, "xxx", endpoint[7:])

	s3client, err := inscription.NewClient(endpoint[7:], &inscription.Options{}, addr, privKey, pubKey)
	if err != nil {
		log.Println("create client fail")
	}

	return s3client, err
}

func ParseBucketAndObject(urlPath string) (bucketName, objectName string) {
	fmt.Println("url path:", urlPath)
	if strings.Contains(urlPath, "s3://") {
		urlPath = urlPath[len("s3://"):]
	}
	splits := strings.SplitN(urlPath, "/", 2)
	fmt.Println("splits:", splits)
	return splits[0], splits[1]
}

func ParseBucket(urlPath string) (bucketName string) {
	if strings.Contains(urlPath, "s3://") {
		urlPath = urlPath[len("s3://"):]
	}
	splits := strings.SplitN(urlPath, "/", 1)

	return splits[0]
}
