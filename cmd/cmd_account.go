package main

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	"github.com/bnb-chain/greenfield/sdk/types"
	gnfdsdktypes "github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdCreatePaymentAccount creates a payment account under the owner
func cmdCreatePaymentAccount() *cli.Command {
	return &cli.Command{
		Name:      "create-account",
		Action:    CreatePaymentAccount,
		Usage:     "create a payment account",
		ArgsUsage: "",
		Description: `
Create a payment account

Examples:
# Create a payment account
$ gnfd-cmd payment create-account `,
	}
}

func CreatePaymentAccount(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	c, createPaymentAccount := context.WithCancel(globalContext)
	defer createPaymentAccount()
	acc, err := client.GetDefaultAccount()
	if err != nil {
		return toCmdErr(err)
	}
	txHash, err := client.CreatePaymentAccount(c, acc.GetAddress().String(), types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	_, err = client.WaitForTx(c, txHash)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("create payment account for %s succ, txHash: %s\n", acc.GetAddress().String(), txHash)
	return nil
}

// cmdPaymentDeposit makes deposit from the owner account to the payment account
func cmdPaymentDeposit() *cli.Command {
	return &cli.Command{
		Name:   "deposit",
		Action: Deposit,
		Usage:  "deposit into stream(payment) account",
		Description: `
Make a deposit into stream(payment) account 

Examples:
# deposit a stream account
$ gnfd-cmd payment deposit --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlag,
				Value:    "",
				Usage:    "the stream account",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlag,
				Value: "",
				Usage: "the amount to be deposited",
			},
		},
	}
}

func Deposit(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	toAddr := ctx.String(toAddressFlag)
	_, err = sdk.AccAddressFromHexUnsafe(toAddr)
	if err != nil {
		return toCmdErr(err)
	}
	amountStr := ctx.String(amountFlag)
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("invalid amount %s", amountStr))
	}
	c, deposit := context.WithCancel(globalContext)
	defer deposit()

	txHash, err := client.Deposit(c, toAddr, amount, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	_, err = client.WaitForTx(c, txHash)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("Deposit %s wei to payment account %s succ, txHash=%s\n", amount.String(), toAddr, txHash)
	return nil
}

// cmdPaymentWithdraw makes a withdrawal from payment account to owner account
func cmdPaymentWithdraw() *cli.Command {
	return &cli.Command{
		Name:   "withdraw",
		Action: Withdraw,
		Usage:  "withdraw from stream(payment) account",
		Description: `
Make a withdrawal from stream(payment) account 

Examples:
# withdraw from a stream account back to the creator account
$ gnfd-cmd payment withdraw --fromAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     fromAddressFlag,
				Value:    "",
				Usage:    "the stream account",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlag,
				Value: "",
				Usage: "the amount to be withdrew",
			},
		},
	}
}

func Withdraw(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	fromAddr := ctx.String(fromAddressFlag)
	_, err = sdk.AccAddressFromHexUnsafe(fromAddr)
	if err != nil {
		return toCmdErr(err)
	}
	amountStr := ctx.String(amountFlag)
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("invalid amount %s", amountStr))
	}
	c, deposit := context.WithCancel(globalContext)
	defer deposit()

	txHash, err := client.Withdraw(c, fromAddr, amount, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	_, err = client.WaitForTx(c, txHash)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("Withdraw %s from %s succ, txHash=%s\n", amount.String(), fromAddr, txHash)
	return nil
}

// cmdListPaymentAccounts list the payment accounts belong to the owner
func cmdListPaymentAccounts() *cli.Command {
	return &cli.Command{
		Name:      "ls",
		Action:    listPaymentAccounts,
		Usage:     "list payment accounts of the owner",
		ArgsUsage: "address of owner",
		Description: `
List payment accounts of the owner.

Examples:
$ gnfd-cmd payment ls `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  ownerAddressFlag,
				Value: "",
				Usage: "indicate a owner's payment accounts to be list, account address can be omitted for current user's accounts listing",
			},
		},
	}
}

func listPaymentAccounts(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	var ownerAddr string
	ownerAddrStr := ctx.String(ownerAddressFlag)
	if ownerAddrStr != "" {
		_, err = sdk.AccAddressFromHexUnsafe(ownerAddrStr)
		if err != nil {
			return toCmdErr(err)
		}
		ownerAddr = ownerAddrStr
	} else {
		acct, err := client.GetDefaultAccount()
		if err != nil {
			return toCmdErr(err)
		}
		ownerAddr = acct.GetAddress().String()
	}
	accounts, err := client.GetPaymentAccountsByOwner(c, ownerAddr)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Println("Accounts not exist")
			return nil
		}
		return toCmdErr(err)
	}
	if len(accounts) == 0 {
		fmt.Println("Accounts not exist")
		return nil
	}
	fmt.Println("payment accounts list:")
	for i, a := range accounts {
		fmt.Printf("%d: %s \n", i+1, a)
	}
	return nil
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
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	var addr string
	flagAddr := ctx.String(addressFlag)
	if flagAddr != "" {
		_, err = sdk.AccAddressFromHexUnsafe(flagAddr)
		if err != nil {
			return toCmdErr(err)
		}
		addr = flagAddr
	} else {
		acct, err := client.GetDefaultAccount()
		if err != nil {
			return toCmdErr(err)
		}
		addr = acct.GetAddress().String()
	}

	resp, err := client.GetAccountBalance(c, addr)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("balance: %s%s\n", resp.Amount.String(), gnfdsdktypes.Denom)
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
				Usage: "the amount to be sent",
			},
		},
	}
}

func Transfer(ctx *cli.Context) error {
	client, err := NewClient(ctx)
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
	_, err = client.WaitForTx(c, txHash)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("transfer %s wei to address %s succ, txHash: %s\n", amountStr, toAddr, txHash)
	return nil
}
