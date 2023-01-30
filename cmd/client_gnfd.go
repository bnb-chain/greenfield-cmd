package main

import (
	"fmt"
	"log"
	"strings"

	greenfield "github.com/bnb-chain/greenfield-sdk-go"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/urfave/cli/v2"
)

// NewClient returns a new greenfield client
func NewClient(ctx *cli.Context) (*greenfield.Client, error) {
	// generate for temp test, it should fetch private key from keystore
	privKey, pubKey, addr := testdata.KeyEthSecp256k1TestPubAddr()

	endpoint := ctx.String("endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("parse endpoint from config file fail")
	}

	if len(endpoint) <= 7 {
		return nil, fmt.Errorf("endpoint length error")
	}
	s3client, err := greenfield.NewClient(endpoint[7:], &greenfield.Options{}, addr, privKey, pubKey)
	if err != nil {
		log.Println("create client fail")
	}

	return s3client, err
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
