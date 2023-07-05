package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/eth/ethsecp256k1"
	"github.com/urfave/cli/v2"
)

// cmdImportAccount import the account by private key file
func cmdImportAccount() *cli.Command {
	return &cli.Command{
		Name:      "import",
		Action:    importKey,
		Usage:     "import the account by the private key file",
		ArgsUsage: "[ <keyfile> ] ",
		Description: `
Import account info from private key file and generate a keystore file to manage user's private key information.
If no keyfile is specified, a keystore will be generated at the default path （homedir/.gnfd-cmd/keystore/key.json）

Examples:
$ gnfd-cmd  account import --privKeyFile  key.txt  ./key.json`,
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

func cmdListAccount() *cli.Command {
	return &cli.Command{
		Name:      "ls",
		Action:    listAccounts,
		Usage:     "list account info",
		ArgsUsage: " ",
		Description: `
list the account info, if the user needs to print the privateKey info, set privateKey flag as true

Examples:
$ gnfd-cmd account ls `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  privKeyFlag,
				Value: "",
				Usage: "include the private key in the output",
			},
		},
	}
}

func cmdCreateAccount() *cli.Command {
	return &cli.Command{
		Name:      "new",
		Action:    createAccount,
		Usage:     "create a new account",
		ArgsUsage: "",
		Description: `
create a new account and store the private key in a keystore file

Examples:
$ gnfd-cmd account new  `,
	}
}

func importKey(ctx *cli.Context) error {
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

	fmt.Printf("import account successfully, key address: %s, encrypted key file: %s \n", key.Address, keyFilePath)

	return nil
}

func listAccounts(ctx *cli.Context) error {
	privateKey, keyfile, err := parseKeystore(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	printPrivate := ctx.Bool(privKeyFlag)

	priBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return toCmdErr(err)
	}

	var keyBytesArray [32]byte
	copy(keyBytesArray[:], priBytes[:32])
	priKey := hd.EthSecp256k1.Generate()(keyBytesArray[:]).(*ethsecp256k1.PrivKey)
	pubKey := priKey.PubKey()

	if !printPrivate {
		fmt.Printf("Account: { %s },  Keystore : %s \n", pubKey.Address(), keyfile)
	} else {
		fmt.Printf("Account: { %s },  Keystore : %s:, Private-key: %s \n", pubKey.Address(), keyfile, privateKey)
	}

	return nil
}

func createAccount(ctx *cli.Context) error {
	keyFilePath := ctx.String("keystore")
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

	account, privateKey, err := sdktypes.NewAccount("gnfd-account")
	if err != nil {
		return toCmdErr(err)
	}

	key := &Key{
		Address:    account.GetAddress(),
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

	fmt.Printf("create new account: {%s} successfully \n", account.GetAddress())
	return nil
}

func parseKeystore(ctx *cli.Context) (string, string, error) {
	keyjson, keyFile, err := loadKeyStoreFile(ctx)
	if err != nil {
		return "", "", toCmdErr(err)
	}

	// fetch password content
	password, err := getPassword(ctx)
	if err != nil {
		return "", "", toCmdErr(err)
	}

	privateKey, err := DecryptKey(keyjson, password)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypting key: %v \n", err)
	}

	return privateKey, keyFile, nil
}
