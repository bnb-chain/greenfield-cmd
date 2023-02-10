package greenfield

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bnb-chain/greenfield-sdk-go/pkg/s3utils"
	"github.com/bnb-chain/greenfield-sdk-go/pkg/signer"
)

// ObjectInfo contain the meta of downloaded objects
type ObjectInfo struct {
	ObjectName  string
	Etag        string
	ContentType string
	Size        int64
}

// GetObjectOptions contains the options of getObject
type GetObjectOptions struct {
	ResponseContentType string `url:"response-content-type,omitempty" header:"-"`
	Range               string `url:"-" header:"Range,omitempty"`
}

// GetObject download s3 object payload and return the related object info
func (c *Client) GetObject(ctx context.Context, bucketName, objectName string, opts GetObjectOptions, authInfo signer.AuthInfo) (io.ReadCloser, ObjectInfo, error) {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return nil, ObjectInfo{}, err
	}
	if err := s3utils.IsValidObjectName(objectName); err != nil {
		return nil, ObjectInfo{}, err
	}

	reqMeta := requestMeta{
		bucketName:    bucketName,
		objectName:    objectName,
		contentSHA256: EmptyStringSHA256,
	}

	//  use for override certain response header values
	//  https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html
	if opts.ResponseContentType != "" {
		urlVal := make(url.Values)
		urlVal["response-content-type"] = []string{opts.ResponseContentType}
		reqMeta.urlValues = urlVal
	}

	if opts.Range != "" {
		reqMeta.Range = opts.Range
	}

	sendOpt := sendOptions{
		method:           http.MethodGet,
		disableCloseBody: true,
	}

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt, authInfo)
	if err != nil {
		log.Printf("get Object %s fail: %s \n", objectName, err.Error())
		return nil, ObjectInfo{}, err
	}

	ObjInfo, err := getObjInfo(bucketName, objectName, resp.Header)
	if err != nil {
		log.Printf("get ObjectInfo %s fail: %s \n", objectName, err.Error())
		closeResponse(resp)
		return nil, ObjectInfo{}, err
	}

	return resp.Body, ObjInfo, nil

}

// FGetObject download s3 object payload adn write the object content into local file specified by filePath
func (c *Client) FGetObject(ctx context.Context, bucketName, objectName string, filePath string, opts GetObjectOptions, authinfo signer.AuthInfo) error {
	// Verify if destination already exists.
	st, err := os.Stat(filePath)
	if err == nil {
		// If the destination exists and is a directory.
		if st.IsDir() {
			return errors.New("fileName is a directory.")
		}
	}

	// If file exist, open it in append mode
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}

	body, _, err := c.GetObject(ctx, bucketName, objectName, opts, authinfo)
	if err != nil {
		log.Printf("download object:%s fail %s \n", objectName, err.Error())
		return err
	}
	defer body.Close()

	_, err = io.Copy(fd, body)
	fd.Close()
	if err != nil {
		return err
	}

	return nil
}

// getObjInfo generate objectInfo base on the response http header content
func getObjInfo(bucketName string, objectName string, h http.Header) (ObjectInfo, error) {
	var etagVal string
	etag := h.Get("Etag")
	if etag != "" {
		etagVal = strings.TrimSuffix(strings.TrimPrefix(etag, "\""), "\"")
	}

	// Parse content length is exists
	var size int64 = -1
	contentLength := h.Get(HTTPHeaderContentLength)
	if contentLength != "" {
		_, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			return ObjectInfo{}, ErrResponse{
				Code:       "InternalError",
				Message:    fmt.Sprintf("Content-Length parse error %v", err),
				BucketName: bucketName,
				ObjectName: objectName,
				RequestID:  h.Get("x-gnfd-request-id"),
			}
		}
	}

	// fetch content type
	contentType := strings.TrimSpace(h.Get("Content-Type"))
	if contentType == "" {
		contentType = contentDefault
	}

	return ObjectInfo{
		ObjectName:  objectName,
		Etag:        etagVal,
		ContentType: contentType,
		Size:        size,
	}, nil

}
