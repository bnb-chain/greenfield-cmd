package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdImportAccount import the account by private key file
func cmdImportAccount() *cli.Command {
	return &cli.Command{
		Name:      "import",
		Action:    importKey,
		Usage:     "import the account by the private key file",
		ArgsUsage: " <privateKeyFile>",
		Description: `
Import account info from private key file and generate a keystore file to manage user's private key information.
If no keyfile is specified by --keystore or -k flag, a keystore will be generated at the default path （homedir/.gnfd-cmd/keystore/key.json）
Users need to set the private key file path which contain the origin private hex string .

Examples:
// key.txt contains the origin private hex string 
$ gnfd-cmd  account import  key.txt `,
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

func cmdExportAccount() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Action:    exportAccount,
		Usage:     "export private key info ",
		ArgsUsage: "",
		Description: `
Export a private key from the local keyring file in a encrypted format.
When both the --unarmored-hex and --unsafe flags are selected, cryptographic
private key material is exported in an INSECURE fashion that is designed to
allow users to import their keys in hot wallets. 

Examples:
$ gnfd-cmd account export --unarmoredHex --unsafe`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  unsafeFlag,
				Usage: "indicate export private key in plain text",
			},
			&cli.BoolFlag{
				Name:  unarmoredFlag,
				Usage: "indicate export private key in plain text",
			},
		},
	}
}

func cmdGetAccountBalance() *cli.Command {
	return &cli.Command{
		Name:      "balance",
		Action:    getAccountBalance,
		Usage:     "query a account's balance",
		ArgsUsage: "",
		Description: `
Get the account balance, if address not specified, default to cur user's account

Examples:
$ gnfd-cmd bank balance --address 0x... `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  addressFlag,
				Value: "",
				Usage: "indicate the address's balance to be retrieved",
			},
		},
	}
}

func getAccountBalance(ctx *cli.Context) error {
	client, err := NewClient(ctx, true)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	addr, err := getUserAddress(ctx)
	if err != nil {
		return err
	}

	resp, err := client.GetAccountBalance(c, addr)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("balance: %s wei%s\n", resp.Amount.String(), types.Denom)
	return nil
}

func cmdTransfer() *cli.Command {
	return &cli.Command{
		Name:      "transfer",
		Action:    Transfer,
		Usage:     "transfer from your account to a dest account",
		ArgsUsage: "",
		Description: `
Make a transfer from your account to a dest account

Examples:
# Create a transfer
$ gnfd-cmd bank transfer --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlag,
				Value:    "",
				Usage:    "the receiver address in BSC",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlag,
				Value: "",
				Usage: "the amount to be sent, the unit is wei for BNB",
			},
		},
	}
}

// cmdBridge makes a transfer from Greenfield to BSC
func cmdBridge() *cli.Command {
	return &cli.Command{
		Name:      "bridge",
		Action:    Bridge,
		Usage:     "transfer from greenfield to a BSC account",
		ArgsUsage: "",
		Description: `
Create a cross chain transfer from Greenfield to a BSC account

Examples:
# Make a cross chain transfer to BSC
$ gnfd-cmd bank bridge --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlag,
				Value:    "",
				Usage:    "the receiver address in BSC",
				Required: true,
			},
			&cli.StringFlag{
				Name:     amountFlag,
				Value:    "",
				Usage:    "the amount of BNB to be sent",
				Required: true,
			},
		},
	}
}

func importKey(ctx *cli.Context) error {
	var (
		err               error
		homeDir, password string
		encryptContent    []byte
	)
	privateKeyFile := ctx.Args().First()
	if privateKeyFile == "" {
		return toCmdErr(errors.New("fail to get the private key file info"))
	}

	// Load private key from file.
	privateKey, addr, err := loadKey(privateKeyFile)
	if err != nil {
		return toCmdErr(errors.New("failed to load private key: %v" + err.Error()))
	}

	homeDir, err = getHomeDir(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	if isKeystoreExist(homeDir+"/"+DefaultKeyDir, addr.String()) {
		return toCmdErr(errors.New("account already exists"))
	}

	keyFilePath := ctx.String("keystore")
	if keyFilePath == "" {
		utcTimestamp := time.Now().UTC().Format(timeFormat)
		keyFilePath = filepath.Join(homeDir, DefaultKeyDir+"/"+utcTimestamp+"--"+convertAddressToLower(addr.String()))
	}

	if _, err = os.Stat(keyFilePath); err == nil {
		return toCmdErr(errors.New("key already exists at :" + keyFilePath))
	} else if !os.IsNotExist(err) {
		return toCmdErr(err)
	}

	key := &Key{
		Address:    addr,
		PrivateKey: privateKey,
	}

	// fetch password content
	password, err = getPassword(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	// encrypt the private key
	encryptContent, err = EncryptKey(key, password, EncryptScryptN, EncryptScryptP)
	if err != nil {
		return toCmdErr(err)
	}

	if err = os.MkdirAll(filepath.Dir(keyFilePath), 0700); err != nil {
		return toCmdErr(errors.New("failed to create directory %s" + filepath.Dir(keyFilePath)))
	}

	// store the keystore file
	if err = os.WriteFile(keyFilePath, encryptContent, 0600); err != nil {
		return toCmdErr(fmt.Errorf("failed to write keyfile to the path%s: %v", keyFilePath, err))
	}

	// if it is the first keystore, set it as the default key
	checkAndWriteDefaultKey(homeDir, convertAddressToLower(key.Address.String()))

	fmt.Printf("key address: %s, encrypted key file: %s \n", key.Address, keyFilePath)
	return nil
}

func listAccounts(ctx *cli.Context) error {
	homeDir, err := getHomeDir(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	keyfile := filepath.Join(homeDir, DefaultKeyDir)
	if err = parseKeyStore(keyfile); err != nil {
		return toCmdErr(err)
	}

	return nil
}

func parseKeyStore(keystoreDir string) error {
	var (
		keyFileContent []byte
		err            error
	)
	files, err := os.ReadDir(keystoreDir)
	if err != nil {
		return errors.New("keystore not exists")
	}

	for _, file := range files {
		if !file.IsDir() {
			// if it is not a valid key file name , bypass it
			if len(file.Name()) < len(timeFormat)+40 || !strings.Contains(file.Name(), "--") {
				continue
			}
			keyPath := filepath.Join(keystoreDir, file.Name())
			keyFileContent, err = os.ReadFile(keyPath)
			if err != nil {
				return fmt.Errorf("failed to read the keyfile at '%s': %v \n", keyPath, err)
			}

			k := new(encryptedKey)
			if err = json.Unmarshal(keyFileContent, k); err != nil {
				return toCmdErr(err)
			}

			fmt.Printf("Account: { %s },  Keystore : %s \n", k.Address, keyPath)
		}
	}
	return nil
}

func exportAccount(ctx *cli.Context) error {
	unsafe := ctx.Bool(unsafeFlag)
	unarmored := ctx.Bool(unarmoredFlag)

	if unarmored && unsafe {
		privateKey, _, err := parseKeystore(ctx)
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Println("Private key: ", privateKey)
		return nil
	} else if unarmored || unsafe {
		return fmt.Errorf("the flags %s and %s must be used together", unsafeFlag, unarmoredFlag)
	}

	keyContent, _, err := loadKeyStoreFile(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	keyJson := new(encryptedKey)
	if err = json.Unmarshal(keyContent, keyJson); err != nil {
		return toCmdErr(err)
	}

	fmt.Println("Armored key: ", keyJson.Crypto.CipherText)

	return nil
}

func createAccount(ctx *cli.Context) error {
	var (
		err            error
		homeDir        string
		password       string
		encryptContent []byte
	)

	account, privateKey, err := sdktypes.NewAccount("gnfd-account")
	if err != nil {
		return toCmdErr(err)
	}

	key := &Key{
		Address:    account.GetAddress(),
		PrivateKey: privateKey,
	}

	homeDir, err = getHomeDir(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	keyFilePath := ctx.String("keystore")
	if keyFilePath == "" {
		utcTimestamp := time.Now().UTC().Format(timeFormat)
		keyFilePath = filepath.Join(homeDir, DefaultKeyDir+"/"+utcTimestamp+"--"+convertAddressToLower(account.GetAddress().String()))
	}

	if _, err = os.Stat(keyFilePath); err == nil {
		return toCmdErr(errors.New("key already exists at :" + keyFilePath))
	} else if !os.IsNotExist(err) {
		return toCmdErr(err)
	}

	// fetch password content
	password, err = getPassword(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	// encrypt the private key
	encryptContent, err = EncryptKey(key, password, EncryptScryptN, EncryptScryptP)
	if err != nil {
		return toCmdErr(err)
	}

	if err = os.MkdirAll(filepath.Dir(keyFilePath), 0700); err != nil {
		return toCmdErr(errors.New("failed to create directory %s" + filepath.Dir(keyFilePath)))
	}
	// store the keystore file
	if err = os.WriteFile(keyFilePath, encryptContent, 0600); err != nil {
		return toCmdErr(fmt.Errorf("failed to write keyfile to the path%s: %v", keyFilePath, err))
	}

	// if it is the first keystore, set it as the default key
	checkAndWriteDefaultKey(homeDir, convertAddressToLower(key.Address.String()))

	fmt.Printf("created new account: {%s} \n", account.GetAddress())
	return nil
}

func Bridge(ctx *cli.Context) error {
	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	c, transfer := context.WithCancel(globalContext)
	defer transfer()

	toAddr := ctx.String(toAddressFlag)
	_, err = sdk.AccAddressFromHexUnsafe(toAddr)
	if err != nil {
		return toCmdErr(err)
	}
	amountStr := ctx.String(amountFlag)
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("%s is not valid amount", amount))
	}
	txResp, err := client.TransferOut(c, toAddr, amount, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txResp.TxHash, "Bridge")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("transfer out %s BNB to %s succ, txHash: %s\n", amountStr, toAddr, txResp.TxHash)
	return nil
}

func Transfer(ctx *cli.Context) error {
	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	c, transfer := context.WithCancel(globalContext)
	defer transfer()

	toAddr := ctx.String(toAddressFlag)
	_, err = sdk.AccAddressFromHexUnsafe(toAddr)
	if err != nil {
		return toCmdErr(err)
	}
	amountStr := ctx.String(amountFlag)
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("%s is not valid amount", amount))
	}
	txHash, err := client.Transfer(c, toAddr, amount, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txHash, "Transfer")
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("transfer %s BNB to address %s succ, txHash: %s\n", amountStr, toAddr, txHash)
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

func checkAndWriteDefaultKey(homeDir string, content string) {
	var err error
	filePath := filepath.Join(homeDir, DefaultAccountPath)
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
			fmt.Printf("failed to create directory %s, error: %v\n", filepath.Dir(filePath), err)
		}

		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			fmt.Printf("write default keystore info fail 123 %v \n", err)
			return
		}
	} else {
		// file exist, check if it is empty
		fileContent, readErr := os.ReadFile(filePath)
		if readErr != nil {
			fmt.Printf("read default keystore info fail %v \n", readErr)
			return
		}

		if len(fileContent) == 0 {
			err = os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				fmt.Printf("write default keystore info fail %v \n", err)
			}
		}
	}
}

func isKeystoreExist(keystoreDir string, address string) bool {
	files, err := os.ReadDir(keystoreDir)
	if err != nil {
		return false
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), convertAddressToLower(address)) {
			filePath := filepath.Join(keystoreDir, file.Name())
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				return false
			}

			if len(fileContent) > 0 {
				return true
			}
		}
	}
	return false
}
