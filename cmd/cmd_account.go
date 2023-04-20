package main

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	gnfdsdktypes "github.com/bnb-chain/greenfield/sdk/types"
	paymenttypes "github.com/bnb-chain/greenfield/x/payment/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdCreatePaymentAccount creates a payment account under the owner
func cmdCreatePaymentAccount() *cli.Command {
	return &cli.Command{
		Name:      "payment-create-account",
		Action:    CreatePaymentAccount,
		ArgsUsage: "",
		Description: `
Create a payment account

Examples:
# Create a payment account
$ gnfd-cmd -c config.toml create-payment-account `,
	}
}

func CreatePaymentAccount(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}

	creator := km.GetAddr().String()
	msg := paymenttypes.NewMsgCreatePaymentAccount(creator)

	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}
	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("create-payment-account for %s failed, txHash=%s\n", txHash))
	}
	fmt.Printf("create-payment-account for %s succ, txHash: %s\n", creator, txHash)
	return nil
}

// cmdPaymentDeposit makes deposit from the owner account to the payment account
func cmdPaymentDeposit() *cli.Command {
	return &cli.Command{
		Name:   "payment-deposit",
		Action: Deposit,
		Usage:  "deposit",
		Description: `
Make a deposit into stream(payment) account 

Examples:
# deposit a stream account
$ gnfd-cmd -c config.toml payment-deposit --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlagName,
				Value:    "",
				Usage:    "the stream account",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
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

	toAddr := ctx.String(toAddressFlagName)
	amountStr := ctx.String(amountFlagName)
	amount, _ := math.NewIntFromString(amountStr)

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("invalid amount %s", amountStr))
	}
	msg := paymenttypes.NewMsgDeposit(
		km.GetAddr().String(),
		toAddr,
		amount,
	)
	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}

	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("Deposit %s BNB to %s failed, txHash=%s\n", amount.String(), toAddr, txHash))
	}
	fmt.Printf("Deposit %s BNB to %s succ, txHash=%s\n", amount.String(), toAddr, txHash)
	return nil
}

// cmdPaymentWithdraw makes a withdrawal from payment account to owner account
func cmdPaymentWithdraw() *cli.Command {
	return &cli.Command{
		Name:   "payment-withdraw",
		Action: Withdraw,
		Usage:  "withdraw",
		Description: `
Make a withdrawal from stream(payment) account 

Examples:
# withdraw from a stream account back to the creator account
$ gnfd-cmd -c config.toml payment-withdraw --fromAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     fromAddressFlagName,
				Value:    "",
				Usage:    "the stream account",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
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

	fromAddr := ctx.String(fromAddressFlagName)
	amountStr := ctx.String(amountFlagName)
	amount, _ := math.NewIntFromString(amountStr)

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("invalid amount %s", amountStr))
	}
	msg := paymenttypes.NewMsgWithdraw(
		km.GetAddr().String(),
		fromAddr,
		amount,
	)
	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}

	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("Withdraw %s from %s failed, txHash=%s\n", amount.String(), fromAddr, txHash))
	}
	fmt.Printf("Withdraw %s from %s succ, txHash=%s\n", amount.String(), fromAddr, txHash)
	return nil
}

// cmdListPaymentAccounts list the payment accounts belong to the owner
func cmdListPaymentAccounts() *cli.Command {
	return &cli.Command{
		Name:      "ls-payment-account",
		Action:    listPaymentAccounts,
		Usage:     "list payment accounts of the owner",
		ArgsUsage: "address of owner",
		Description: `
List payment accounts of the owner.

Examples:
$ gnfd-cmd -c config.toml ls-payment-account `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  ownerAddressFlagName,
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
	ownerAddrStr := ctx.String(ownerAddressFlagName)
	if ownerAddrStr != "" {
		ownerAddr = ownerAddrStr
	} else {
		km, err := client.ChainClient.GetKeyManager()
		if err != nil {
			return toCmdErr(err)
		}
		ownerAddr = km.GetAddr().String()
	}

	req := paymenttypes.QueryGetPaymentAccountsByOwnerRequest{Owner: ownerAddr}

	accounts, err := client.ChainClient.PaymentQueryClient.GetPaymentAccountsByOwner(c, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Println("Accounts not exist")
			return nil
		}
		return toCmdErr(err)
	}

	if len(accounts.PaymentAccounts) == 0 {
		fmt.Println("Accounts not exist")
		return nil
	}

	fmt.Println("payment accounts list:")
	for i, a := range accounts.PaymentAccounts {
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
$ gnfd-cmd -c config.toml balance --address 0x... `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  addressFlagName,
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
	flagAddr := ctx.String(addressFlagName)
	if flagAddr != "" {
		addr = flagAddr
	} else {
		km, err := client.ChainClient.GetKeyManager()
		if err != nil {
			return toCmdErr(err)
		}
		addr = km.GetAddr().String()
	}

	req := banktypes.QueryBalanceRequest{Address: addr,
		Denom: gnfdsdktypes.Denom}

	resp, err := client.ChainClient.BankQueryClient.Balance(c, &req)
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("balance: %s%s\n", resp.Balance.Amount.String(), gnfdsdktypes.Denom)
	return nil
}

func cmdTransfer() *cli.Command {
	return &cli.Command{
		Name:      "transfer",
		Action:    Transfer,
		Usage:     "transfer",
		ArgsUsage: "",
		Description: `
Make a transfer from your account to a dest account

Examples:
# Create a transfer
$ gnfd-cmd -c config.toml transfer --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlagName,
				Value:    "",
				Usage:    "the receiver address in BSC",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
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

	toAddr := ctx.String(toAddressFlagName)
	amountStr := ctx.String(amountFlagName)
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("%s is not valid amount", amount))
	}

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}
	to, err := sdk.AccAddressFromHexUnsafe(toAddr)
	if err != nil {
		return toCmdErr(err)
	}

	msg := banktypes.NewMsgSend(
		km.GetAddr(),
		to,
		sdk.NewCoins(sdk.NewCoin(gnfdsdktypes.Denom, amount)),
	)

	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}
	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("transfer %s BNB to %s failed, txHash=%s\n", amountStr, toAddr, txHash))

	}
	fmt.Printf("transfer %s BNB to %s succ, txHash: %s\n", amountStr, toAddr, txHash)
	return nil
}
