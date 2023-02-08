package greenfield

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/bnb-chain/greenfield-sdk-go/pkg/signer"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PutObjectMeta  represents meta which is used to construct PutObjectMsg
type PutObjectMeta struct {
	PaymentAccount sdk.AccAddress
	PrimarySp      string
	IsPublic       bool
	ObjectSize     int64
	ContentType    string
}

// ObjectMeta represents meta which may needed when upload payload
type ObjectMeta struct {
	ObjectSize  int64
	ContentType string
}

// UploadResult contains information about the object which has been upload
type UploadResult struct {
	BucketName string
	ObjectName string
	ETag       string // Hex encoded unique entity tag of the object.
}

func (t *UploadResult) String() string {
	return fmt.Sprintf("upload finish, bucket name  %s, objectname %s, etag %s", t.BucketName, t.ObjectName, t.ETag)
}

// PrePutObject get approval of creating object and send txn to greenfield chain
func (c *Client) PrePutObject(ctx context.Context, bucketName, objectName string,
	meta PutObjectMeta, reader io.Reader, authInfo signer.AuthInfo) (string, error) {
	// get approval of creating bucket from sp
	signature, err := c.GetApproval(ctx, bucketName, objectName, authInfo)
	if err != nil {
		return "", err
	}
	log.Println("get approve from sp finish,signature is: ", signature)

	// get hash and objectSize from reader
	_, _, err = SplitAndComputerHash(reader, SegmentSize, EncodeShards)
	if err != nil {
		return "", err
	}

	// TODO(leo) call chain sdk to send a createObject txn to greenfield
	// return txnHash

	return "", err
}

// PutObject supports the second stage of uploading the object to bucket.
func (c *Client) PutObject(ctx context.Context, bucketName, objectName, txnHash string,
	reader io.Reader, meta ObjectMeta, authInfo signer.AuthInfo) (res UploadResult, err error) {
	if txnHash == "" {
		return UploadResult{}, errors.New("txn hash empty")
	}

	if meta.ObjectSize <= 0 {
		return UploadResult{}, errors.New("object size not set")
	}

	if meta.ContentType == "" {
		return UploadResult{}, errors.New("content type not set")
	}

	reqMeta := requestMeta{
		bucketName:    bucketName,
		objectName:    objectName,
		contentSHA256: EmptyStringSHA256,
		contentLength: meta.ObjectSize,
		contentType:   meta.ContentType,
	}

	sendOpt := sendOptions{
		method:  http.MethodPut,
		body:    reader,
		txnHash: txnHash,
	}

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt, authInfo)
	if err != nil {
		log.Printf("upload payload the object failed: %s \n", err.Error())
		return UploadResult{}, err
	}

	etagValue := resp.Header.Get(HTTPHeaderEtag)

	return UploadResult{
		BucketName: bucketName,
		ObjectName: objectName,
		ETag:       etagValue,
	}, nil
}

// FPutObject support upload object from local file
func (c *Client) FPutObject(ctx context.Context, bucketName, objectName,
	filePath, txnHash, contentType string, authInfo signer.AuthInfo) (res UploadResult, err error) {
	fReader, err := os.Open(filePath)
	// If any error fail quickly here.
	if err != nil {
		return UploadResult{}, err
	}
	defer fReader.Close()

	// Save the file stat.
	stat, err := fReader.Stat()
	if err != nil {
		return UploadResult{}, err
	}

	meta := ObjectMeta{
		ObjectSize: stat.Size(),
	}
	
	if contentType == "" {
		meta.ContentType = "application/octet-stream"
	} else {
		meta.ContentType = contentType
	}

	return c.PutObject(ctx, bucketName, objectName, txnHash, fReader, meta, authInfo)
}
