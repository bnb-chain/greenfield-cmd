package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func generateKey(ctx *cli.Context) error {
	keyFilePath := ctx.Args().First()
	if keyFilePath == "" {
		homeDirname, err := os.UserHomeDir()
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
		return err
	}

	// write password content to default password file path
	err = writeDefaultPassword(password)
	if err != nil {
		return err
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

func writeDefaultPassword(password string) error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return toCmdErr(err)
	}

	filePath := filepath.Join(dirname, DefaultPasswordPath)

	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return toCmdErr(errors.New("failed to create password directory :%s" + filepath.Dir(filePath)))
	}

	// store the password
	if err := os.WriteFile(filePath, []byte(password), 0600); err != nil {
		return toCmdErr(fmt.Errorf("failed to write password to the path: %s: %v", filePath, err))
	}

	fmt.Printf("generate password file: %s successfully \n", filePath)
	return nil
}

func parseKeystore(ctx *cli.Context) (string, error) {
	keyjson, err := loadKeyStoreFile(ctx)
	if err != nil {
		return "", toCmdErr(err)
	}

	var password string
	if passwordFile := ctx.String(passwordFileFlag); passwordFile != "" {
		// load password from password flag
		readContent, err := os.ReadFile(passwordFile)
		if err != nil {
			return "", errors.New("failed to read password file" + err.Error())
		}
		password = strings.TrimRight(string(readContent), "\r\n")
	} else {
		// load password from default password file path
		password, err = loadPassWordFile(ctx)
		if err != nil {
			return "", toCmdErr(err)
		}
	}

	privateKey, err := DecryptKey(keyjson, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypting key: %v \n", err)
	}

	return privateKey, nil
}
