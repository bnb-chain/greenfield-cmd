package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/eth/ethsecp256k1"
	"github.com/urfave/cli/v2"
)

// cmdGenerateKey generate keystore file
func cmdGenerateKey() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Action:    generateKey,
		Usage:     "create a new keystore file",
		ArgsUsage: "[ <keyfile> ] ",
		Description: `
generate a keystore file to manage user's private key information.
Examples:
$ gnfd-cmd keystore generate --privKeyFile key.txt  `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     privKeyFileFlag,
				Value:    "",
				Usage:    "the private key file path which contain the origin private hex string",
				Required: true,
			},
			&cli.StringFlag{
				Name:  passwordFileFlag,
				Value: "",
				Usage: "the file which contains the password for the keyfile",
			},
		},
	}
}

func cmdPrintKey() *cli.Command {
	return &cli.Command{
		Name:      "inspect",
		Action:    inspectKey,
		Usage:     "inspect a keystore file",
		ArgsUsage: "[ <keyfile> ] ",
		Description: `
print the private key related information

Examples:
$ gnfd-cmd  keystore inspect --privateKey true  `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  privKeyFlag,
				Value: "",
				Usage: "include the private key in the output",
			},
			&cli.StringFlag{
				Name:  passwordFileFlag,
				Value: "",
				Usage: "the file which contains the password for the keyfile",
			},
		},
	}
}

func generateKey(ctx *cli.Context) error {
	keyFilePath := ctx.Args().First()
	if keyFilePath == "" {
		homeDirname, err := getHomeDir(ctx)
		if err != nil {
			return toCmdErr(err)
		}
		keyFilePath = filepath.Join(homeDirname, DefaultKeyStorePath)
	}

	if _, err := os.Stat(keyFilePath); err == nil {
		return toCmdErr(errors.New("key already exists at :" + keyFilePath))
	} else if !os.IsNotExist(err) {
		return toCmdErr(err)
	}

	privKeyFile := ctx.String(privKeyFileFlag)
	if privKeyFile == "" {
		return toCmdErr(errors.New("fail to get private key file path, please set it by --privKeyFile"))
	}

	// Load private key from file.
	privateKey, addr, err := loadKey(privKeyFile)
	if err != nil {
		return toCmdErr(errors.New("failed to load private key: %v" + err.Error()))
	}

	key := &Key{
		Address:    addr,
		PrivateKey: privateKey,
	}

	// fetch password content
	password, err := getPassword(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	// encrypt the private key
	encryptContent, err := EncryptKey(key, password, EncryptScryptN, EncryptScryptP)
	if err != nil {
		return toCmdErr(err)
	}

	if err := os.MkdirAll(filepath.Dir(keyFilePath), 0700); err != nil {
		return toCmdErr(errors.New("failed to create directory %s" + filepath.Dir(keyFilePath)))
	}

	// store the keystore file
	if err := os.WriteFile(keyFilePath, encryptContent, 0600); err != nil {
		return toCmdErr(fmt.Errorf("failed to write keyfile to the path%s: %v", keyFilePath, err))
	}

	fmt.Printf("generate keystore %s successfully, key address: %s \n", keyFilePath, key.Address)

	return nil
}

func inspectKey(ctx *cli.Context) error {
	privateKey, err := parseKeystore(ctx)
	if err != nil {
		return nil
	}
	printPrivate := ctx.Bool(privKeyFlag)

	priBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return err
	}

	var keyBytesArray [32]byte
	copy(keyBytesArray[:], priBytes[:32])
	priKey := hd.EthSecp256k1.Generate()(keyBytesArray[:]).(*ethsecp256k1.PrivKey)
	pubKey := priKey.PubKey()

	fmt.Println("Address:       ", pubKey.Address())
	fmt.Println("Public key:    ", pubKey.String())
	if printPrivate {
		fmt.Println("Private key:   ", privateKey)
	}

	return nil
}

func parseKeystore(ctx *cli.Context) (string, error) {
	keyjson, err := loadKeyStoreFile(ctx)
	if err != nil {
		return "", toCmdErr(err)
	}

	// fetch password content
	password, err := getPassword(ctx)
	if err != nil {
		return "", toCmdErr(err)
	}

	privateKey, err := DecryptKey(keyjson, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypting key: %v \n", err)
	}

	return privateKey, nil
}
