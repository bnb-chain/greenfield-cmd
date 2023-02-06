package greenfield

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/bnb-chain/greenfield-sdk-go/pkg/s3utils"
	"github.com/bnb-chain/greenfield-sdk-go/pkg/signer"
)

type ListObjectsResult struct {
	objectList []ObjectInfo
	prefix     string
}

// CreateBucket get approval of creating bucket and send createBucket txn to greenfield chain
func (c *Client) CreateBucket(ctx context.Context, bucketName string, authInfo signer.AuthInfo) error {
	// get approval of creating bucket from sp
	signature, err := c.GetApproval(ctx, bucketName, "", authInfo)
	if err != nil {
		return err
	}

	log.Println("get approve from sp finish,signature is:", signature)
	// TODO(leo) call chain sdk to send a createBucket txn to greenfield with signature

	return nil
}

// ListObjects return object name list of the specific bucket
func (c *Client) ListObjects(ctx context.Context, bucketName, objectPrefix string, maxkeys int, authInfo signer.AuthInfo) (ListObjectsResult, error) {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return ListObjectsResult{}, err
	}

	reqMeta := requestMeta{
		bucketName:    bucketName,
		contentSHA256: EmptyStringSHA256,
	}

	sendOpt := sendOptions{
		method:           http.MethodGet,
		disableCloseBody: true,
	}
	urlValues := make(url.Values)
	urlValues.Set("prefix", objectPrefix)
	if maxkeys > 0 {
		urlValues.Set("max-keys", fmt.Sprintf("%d", maxkeys))
	}
	reqMeta.urlValues = urlValues

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt, authInfo)
	if err != nil {
		log.Printf("listObjects of bucket %s fail: %s \n", bucketName, err.Error())
		return ListObjectsResult{}, err
	}
	defer closeResponse(resp)

	listObjectsResult := ListObjectsResult{}
	// decode the xml content from response body
	err = xml.NewDecoder(resp.Body).Decode(&listObjectsResult)
	if err != nil {
		return ListObjectsResult{}, err
	}

	return listObjectsResult, nil
}
