package main

import (
	"fmt"
	"os"
	"path/filepath"

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
$ gnfd-cmd keystore generate --privKeyFile key.txt  key.json `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     privKeyFileFlag,
				Value:    "",
				Usage:    "the private key file path which contain the origin private hex string",
				Required: true,
			},
		},
	}
}

func generateKey(ctx *cli.Context) error {
	keyFilePath := ctx.Args().First()
	if keyFilePath == "" {
		keyFilePath = defaultKeyfile
	}

	if _, err := os.Stat(keyFilePath); err == nil {
		return genCmdErr("key already exists at :" + keyFilePath)
	} else if !os.IsNotExist(err) {
		return toCmdErr(err)
	}

	if err := os.MkdirAll(filepath.Dir(keyFilePath), 0700); err != nil {
		return genCmdErr("failed to create directory %s" + filepath.Dir(keyFilePath))
	}

	privKeyFile := ctx.String(privKeyFileFlag)
	if privKeyFile == "" {
		return genCmdErr("private key file path")
	}

	// Load private key from file.
	privateKey, addr, err := loadKey(privKeyFile)
	if err != nil {
		return genCmdErr("Can't load private key: %v" + err.Error())
	}

	key := &Key{
		Address:    addr,
		PrivateKey: privateKey,
	}

	configFile := ctx.String("config")
	var config *cmdConfig

	if configFile != "" {
		config, err = parseConfigFile(configFile)
		if err != nil {
			return err
		}
	}

	// fetch password content
	password, err := getPassword(ctx, config)
	if err != nil {
		return err
	}

	// encrypt the private key
	encryptContent, err := EncryptKey(key, password, EncryptScryptN, EncryptScryptP)
	if err != nil {
		return genCmdErr("failed to encrypting key: " + err.Error())
	}

	// store the keystore file
	if err := os.WriteFile(keyFilePath, encryptContent, 0600); err != nil {
		return genCmdErr(fmt.Sprintf("failed to write keyfile to the path%s: %v", keyFilePath, err))
	}

	fmt.Printf("generate keystore %s successfully, key address: %s \n", keyFilePath, key.Address)

	return nil
}
