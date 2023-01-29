package greenfield

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/bnb-chain/bfs/x/storage/types"
	"github.com/bnb-chain/greenfield-sdk-go/pkg/s3utils"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

const PutObjectUrlTxn = "putObjectV2"

// PutObjectOptions represents options specified by user for PutObject call
type PutObjectOptions struct {
	SecondarySp []string
	PartSize    uint64
	replicaNum  int
}

// PutObjectOptions represents meta which is used to construct PutObjectMsg
type PutObjectMeta struct {
	PaymentAccount sdk.AccAddress
	PrimarySp      string
	IsPublic       bool
	ObjectSize     int64
	ContentType    string
	Sha256Hash     string
}

// ObjectMeta represents meta which is needed when upload payload
type ObjectMeta struct {
	ObjectSize  int64
	ContentType string
	Sha256Hash  string
	TxnHash     string
}

// UploadResult contains information about the object which has been upload
type UploadResult struct {
	BucketName string
	ObjectName string
	ETag       string // Hex encoded unique entity tag of the object.
}

// TxnInfo indicates the detail of sent txn info
type TxnInfo struct {
	txnHash       []byte
	createTxnDate time.Time
}

func (t *TxnInfo) String() string {
	return fmt.Sprintf("send txn hash: %s, create time %s", t.txnHash, t.createTxnDate.String())
}

func (t *UploadResult) String() string {
	return fmt.Sprintf("upload finish, bucket name  %s, objectname %s, etag %s", t.BucketName, t.ObjectName, t.ETag)
}

// SendPutObjectTxn supports the first stage of uploading the object to bucket
// The payload of object will not be uploaded at the first stage
// The content-type, object size and sha256hash should be set in the meta
func (c *Client) SendPutObjectTxn(ctx context.Context, bucketName, objectName string,
	meta PutObjectMeta) (TxnInfo, error) {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return TxnInfo{}, err
	}
	if err := s3utils.IsValidObjectName(objectName); err != nil {
		return TxnInfo{}, err
	}

	if meta.ObjectSize < 0 {
		return TxnInfo{}, errors.New("objectSize should not be less than zero")
	}

	if meta.ContentType == "" {
		return TxnInfo{}, errors.New("content type empty")
	}

	if meta.Sha256Hash == "" {
		return TxnInfo{}, errors.New("sha256 hash empty")
	}

	reqMeta := requestMeta{
		bucketName:        bucketName,
		objectName:        objectName,
		gnfdContentLength: meta.ObjectSize,
		contentSHA256:     meta.Sha256Hash,
		contentType:       meta.ContentType,
	}

	sendOpt := sendOptions{
		method:           http.MethodPut,
		disableCloseBody: true,
	}

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt)
	if err != nil {
		log.Printf("send putObjectMsg fail: %s \n", err.Error())
		return TxnInfo{}, err
	}

	// get the transaction hash which is generated after SP has co-signed the txn
	txnHash := resp.Header.Get(HTTPHeaderTransactionHash)
	if txnHash == "" {
		return TxnInfo{}, errors.New("fail to fetch txn hash info")
	}

	txnDate := resp.Header.Get(HTTPHeaderTransactionDate)
	if txnDate == "" {
		return TxnInfo{}, errors.New("fail to fetch txn date")
	}

	createDate, _ := time.Parse("2006-01-02T15:04:05.000Z", txnDate)

	return TxnInfo{txnHash: []byte(txnHash),
		createTxnDate: createDate}, nil
}

// PutObjectWithTxn supports the second stage of uploading the object to bucket.
func (c *Client) PutObjectWithTxn(ctx context.Context, bucketName, objectName string,
	reader io.Reader, meta ObjectMeta) (res UploadResult, err error) {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return UploadResult{}, err
	}
	if err := s3utils.IsValidObjectName(objectName); err != nil {
		return UploadResult{}, err
	}

	if meta.ObjectSize < 0 {
		return UploadResult{}, errors.New("objectSize should not be less than zero")
	}

	if meta.ContentType == "" {
		return UploadResult{}, errors.New("content type empty")
	}

	if meta.Sha256Hash == "" {
		return UploadResult{}, errors.New("sha256 hash empty")
	}

	urlVal := make(url.Values)
	urlVal[PutObjectUrlTxn] = []string{""}

	reqMeta := requestMeta{
		bucketName:        bucketName,
		objectName:        objectName,
		urlValues:         urlVal,
		contentLength:     meta.ObjectSize,
		contentType:       meta.ContentType,
		gnfdContentLength: meta.ObjectSize,
		contentSHA256:     meta.Sha256Hash,
	}

	sendOpt := sendOptions{
		method:  http.MethodPut,
		body:    reader,
		txnHash: meta.TxnHash,
	}

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt)
	if err != nil {
		log.Printf("the second stage of uploading the object failed: %s \n", err.Error())
		return UploadResult{}, err
	}

	etagValue := resp.Header.Get(HTTPHeaderEtag)

	return UploadResult{
		BucketName: bucketName,
		ObjectName: objectName,
		ETag:       etagValue,
	}, nil
}

// genPutObjectMsg construct the createObjectMsg  and sign the msg
func (c *Client) genPutObjectMsg(bucketName, objectName, contentType string, isPublic bool, ObjectSize int64, hashInfo [][]byte) ([]byte, error) {
	createObjectMsg := types.NewMsgCreateObject(
		c.GetAccount(),
		bucketName,
		objectName,
		uint64(ObjectSize),
		isPublic,
		hashInfo,
		contentType,
		[]byte(""),
		[]sdk.AccAddress{nil},
	)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &types.MsgCreateObject{})
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	txConfig := authtx.NewTxConfig(marshaler,
		[]signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_EIP_712})
	txBuilder := txConfig.NewTxBuilder()
	txBuilder.SetMsgs(createObjectMsg)

	// sign the createObjectMsg
	msgBytes, err := c.signer.GetTxnSignBytes(txBuilder)
	if err != nil {
		log.Print("sign put object transaction msg fail", err)
		return []byte(""), err
	}

	return msgBytes, err
}
