package sign

import (
	"fmt"
	"log"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/stretchr/testify/require"
)

func TestSigner(t *testing.T) {
	privKey, pubkey, addr := testdata.KeyEthSecp256k1TestPubAddr()

	sdkAddr := sdk.AccAddress(pubkey.Address())
	fmt.Println("private key:", privKey.String())
	fmt.Println("public key:", pubkey.String())

	testSigner, err := NewSigner(sdkAddr, "/Users/user/.gnfd/", privKey, pubkey)
	require.NoError(t, err)

	_, _, feePayerAddr := testdata.KeyEthSecp256k1TestPubAddr()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &testdata.TestMsg{})
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	txConfig := authtx.NewTxConfig(marshaler, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_EIP_712})
	txBuilder := txConfig.NewTxBuilder()
	memo := "some test memo"

	msgs := []sdk.Msg{banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom,
		sdkmath.NewInt(1))))}

	fee := txtypes.Fee{Amount: sdk.NewCoins(sdk.NewInt64Coin("atom", 150)), GasLimit: 20000}
	err = txBuilder.SetMsgs(msgs...)
	require.NoError(t, err)
	txBuilder.SetMemo(memo)
	txBuilder.SetFeeAmount(fee.Amount)
	txBuilder.SetFeePayer(feePayerAddr)
	txBuilder.SetGasLimit(fee.GasLimit)
	// sign the test txn
	sigV2, err := testSigner.SignTxn(txBuilder)
	require.NoError(t, err)
	log.Printf("signature is: %s", sigV2.Data)

	// verify the signature which signed by the signer
	err = signing.VerifySignature(pubkey, testSigner.GetSignerData(), sigV2.Data,
		testSigner.clientCtx.TxConfig.SignModeHandler(), txBuilder.GetTx())
	require.NoError(t, err)

}
