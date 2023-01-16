package greenfield

import (
	"context"
	"log"
	"net/http"

	"github.com/bnb-chain/bfs/x/storage/types"
	"github.com/bnb-chain/greenfield-sdk-go/pkg/s3utils"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
)

// CreateBucket create a new bucket with the createBucketTxn sent to chain
func (c *Client) CreateBucket(ctx context.Context, bucketName string, isPublic bool) error {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return err
	}
	// Create Bucket request metadata.
	reqMeta := requestMeta{
		bucketName:    bucketName,
		contentSHA256: EmptyStringSHA256,
	}
	// generate sp address for temp
	_, _, spAddr := testdata.KeyEthSecp256k1TestPubAddr()
	msgBytes, err := c.genCreateBucketMsg(bucketName, isPublic, spAddr)
	if err != nil {
		log.Print("generate signed create bucket transaction msg fail", err)
		return err
	}

	sendOpt := sendOptions{
		method: http.MethodPut,
		txnMsg: common.Bytes2Hex(msgBytes),
	}

	_, err = c.sendReq(ctx, reqMeta, &sendOpt)
	if err != nil {
		log.Printf("create bucket fail: %s \n", err.Error())
		return err
	}

	return nil
}

// genCreateBucketMsg construct the genCreateBucketMsg and sign the msg
func (c *Client) genCreateBucketMsg(bucketName string, isPublic bool, primarySP sdk.AccAddress) ([]byte, error) {
	// construct createBucketMsg
	createBucketMsg := types.NewMsgCreateBucket(
		c.GetAccount(),
		bucketName,
		isPublic,
		primarySP,
		primarySP,
		[]byte("test signature"),
	)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &types.MsgCreateBucket{})
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	txConfig := authtx.NewTxConfig(marshaler,
		[]signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_EIP_712})
	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetMsgs(createBucketMsg)

	// sign the createBucketMsg
	msgBytes, err := c.signer.GetTxnSignBytes(txBuilder)
	if err != nil {
		log.Print("sign create bucket transaction msg fail", err)
		return []byte(""), err
	}

	return msgBytes, err
}
