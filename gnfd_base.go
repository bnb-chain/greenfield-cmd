package greenfield

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/greenfield-sdk-go/pkg/s3utils"
	"github.com/bnb-chain/greenfield-sdk-go/pkg/signer"
)

// Client is a client manages communication with the inscription API.
type Client struct {
	endpoint  *url.URL // Parsed endpoint url provided by the user.
	client    *http.Client
	userAgent string
	host      string

	conf    *CliConfig
	sender  sdk.AccAddress      // sender greenfield chain address
	privKey cryptotypes.PrivKey // sender private key
}

// CliConfig is the config info of client
type CliConfig struct {
	Secure           bool // use https or not
	Transport        http.RoundTripper
	RetryOpt         RetryOptions
	UploadLimitSpeed uint64
}

type Options struct {
	secure bool
}

type RetryOptions struct {
	Count      int
	Interval   time.Duration
	StatusCode []int
}

// NewClient returns a new greenfield client
func NewClient(endpoint string, opts *Options) (*Client, error) {
	url, err := getEndpointURL(endpoint, opts.secure)
	if err != nil {
		log.Println("get url error:", err.Error())
		return nil, err
	}

	httpClient := &http.Client{}
	c := &Client{
		client:    httpClient,
		userAgent: UserAgent,
		endpoint:  url,
		conf: &CliConfig{
			RetryOpt: RetryOptions{
				Count:    3,
				Interval: time.Duration(0),
			},
		},
	}

	return c, nil
}

// GetURL returns the URL of the S3 endpoint.
func (c *Client) GetURL() *url.URL {
	endpoint := *c.endpoint
	return &endpoint
}

// requestMeta - contain the metadata to construct the http request.
type requestMeta struct {
	bucketName       string
	objectName       string
	urlRelPath       string     // relative path of url
	urlValues        url.Values // url values to be added into url
	Range            string
	ApproveAction    string
	SignType         string
	contentType      string
	contentLength    int64
	contentMD5Base64 string // base64 encoded md5sum
	contentSHA256    string // hex encoded sha256sum
}

// sendOptions -  options to use to send the http message
type sendOptions struct {
	method           string      // request method
	body             interface{} // request body
	result           interface{} // unmarshal message of the resp.Body
	disableCloseBody bool        // indicate whether to disable automatic calls to resp.Body.Close()
	txnHash          string      // the transaction hash info
	isAdminApi       bool        // indicate if it is an admin api request
}

// SetHost set host name of request
func (c *Client) SetHost(hostName string) {
	c.host = hostName
}

// SetPriKey set private key of client
// it is needed to be set when dapp sign the request using private key
func (c *Client) SetPriKey(key cryptotypes.PrivKey) {
	c.privKey = key
}

// GetHost get host name of request
func (c *Client) GetHost() string {
	return c.host
}

// GetAccount get sender address info
func (c *Client) GetAccount() sdk.AccAddress {
	return c.sender
}

// GetAgent get agent name
func (c *Client) GetAgent() string {
	return c.userAgent
}

// newRequest construct the http request, set url, body and headers
func (c *Client) newRequest(ctx context.Context,
	method string, meta requestMeta, body interface{}, txnHash string, isAdminAPi bool, authInfo signer.AuthInfo) (req *http.Request, err error) {
	// construct the target url
	desURL, err := c.generateURL(meta.bucketName, meta.objectName, meta.urlRelPath, meta.urlValues, isAdminAPi)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	contentType := ""
	sha256Hex := ""
	if body != nil {
		// the body content is io.Reader type
		if ObjectReader, ok := body.(io.Reader); ok {
			reader = ObjectReader
			if meta.contentType == "" {
				contentType = contentDefault
			}
		} else {
			// the body content is xml type
			content, err := xml.Marshal(body)
			if err != nil {
				return nil, err
			}
			contentType = contentTypeXML
			reader = bytes.NewReader(content)
			sha256Hex = CalcSHA256Hex(content)
		}
	}

	// Initialize a new HTTP request for the method.
	req, err = http.NewRequestWithContext(ctx, method, desURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// need to turn the body into ReadCloser
	if body == nil {
		req.Body = nil
	} else {
		req.Body = io.NopCloser(reader)
	}

	// set content length
	req.ContentLength = meta.contentLength

	// set txn hash header
	if txnHash != "" {
		req.Header.Set(HTTPHeaderTransactionHash, txnHash)
	}

	// set content type header
	if meta.contentType != "" {
		req.Header.Set(HTTPHeaderContentType, meta.contentType)
	} else if contentType != "" {
		req.Header.Set(HTTPHeaderContentType, contentType)
	} else {
		req.Header.Set(HTTPHeaderContentType, contentDefault)
	}

	// set md5 header
	if meta.contentMD5Base64 != "" {
		req.Header[HTTPHeaderContentMD5] = []string{meta.contentMD5Base64}
	}

	// set sha256 header
	if meta.contentSHA256 != "" {
		req.Header[HTTPHeaderContentSHA256] = []string{meta.contentSHA256}
	} else {
		req.Header[HTTPHeaderContentSHA256] = []string{sha256Hex}
	}

	if meta.Range != "" && method == http.MethodGet {
		req.Header.Set(HTTPHeaderRange, meta.Range)
	}

	// set request host
	if c.host != "" {
		req.Host = c.host
	} else if req.URL.Host != "" {
		req.Host = req.URL.Host
	}

	if isAdminAPi {
		if meta.objectName == "" {
			req.Header.Set(HTTPHeaderResource, meta.bucketName)
		} else {
			req.Header.Set(HTTPHeaderResource, meta.bucketName+"/"+meta.objectName)
		}
	}

	// set date header
	stNow := time.Now().UTC()
	req.Header.Set(HTTPHeaderDate, stNow.Format(iso8601DateFormatSecond))

	// set user-agent
	req.Header.Set(HTTPHeaderUserAgent, c.userAgent)

	// sign the total http request info when auth type v1
	if authInfo.SignType == signer.AuthV1 && c.privKey != nil {
		err = signer.SignRequest(req, c.privKey, authInfo)
		if err != nil {
			return req, err
		}
	}

	return
}

// doAPI call client.Do() to send request and read response from servers
func (c *Client) doAPI(ctx context.Context, req *http.Request, meta requestMeta, closeBody bool) (*http.Response, error) {
	var cancel context.CancelFunc
	if closeBody {
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if urlErr, ok := err.(*url.Error); ok {
			if strings.Contains(urlErr.Err.Error(), "EOF") {
				return nil, &url.Error{
					Op:  urlErr.Op,
					URL: urlErr.URL,
					Err: errors.New("Connection closed by foreign host " + urlErr.URL + ". Retry again."),
				}
			}
		}
		return nil, err
	}
	defer func() {
		if closeBody {
			closeResponse(resp)
		}
	}()

	// construct err responses and messages
	err = constructErrResponse(resp, meta.bucketName, meta.objectName)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// sendReq new restful request, send the message and handle the response
func (c *Client) sendReq(ctx context.Context, metadata requestMeta, opt *sendOptions, authInfo signer.AuthInfo) (res *http.Response, err error) {
	req, err := c.newRequest(ctx, opt.method, metadata, opt.body, opt.txnHash, opt.isAdminApi, authInfo)
	if err != nil {
		log.Printf("new request error: %s , stop send request\n", err.Error())
		return nil, err
	}

	resp, err := c.doAPI(ctx, req, metadata, !opt.disableCloseBody)
	if err != nil {
		log.Printf("do api request fail: %s \n", err.Error())
		return nil, err
	}
	return resp, nil
}

// genURL construct the target request url based on the parameters
func (c *Client) generateURL(bucketName string, objectName string, relativePath string,
	queryValues url.Values, isAdminApi bool) (*url.URL, error) {
	host := c.endpoint.Host
	// Save scheme.
	scheme := c.endpoint.Scheme

	if h, p, err := net.SplitHostPort(host); err == nil {
		if scheme == "http" && p == "80" || scheme == "https" && p == "443" {
			host = h
			if ip := net.ParseIP(h); ip != nil && ip.To16() != nil {
				host = "[" + h + "]"
			}
		}
	}

	if bucketName == "" {
		err := errors.New("no bucketName in path")
		return nil, err
	}

	var urlStr string
	if isAdminApi {
		prefix := AdminURLPrefix + AdminURLVersion
		urlStr = scheme + "://" + host + prefix + "/"
	} else {
		// generate s3 virtual hosted style url
		if CheckDomainName(host) {
			urlStr = scheme + "://" + bucketName + "." + host + "/"
		} else {
			urlStr = scheme + "://" + host + "/"
		}
		if objectName != "" {
			urlStr += s3utils.EncodePath(objectName)
		}
	}

	if relativePath != "" {
		urlStr += s3utils.EncodePath(relativePath)
	}

	if len(queryValues) > 0 {
		urlStrNew, err := addQueryValues(urlStr, queryValues)
		if err != nil {
			return nil, err
		}
		urlStr = urlStrNew
	}

	return url.Parse(urlStr)
}

// GetApproval return the signature info for the approval of preCreating resources
func (c *Client) GetApproval(ctx context.Context, bucketName, objectName string, authInfo signer.AuthInfo) (string, error) {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return "", err
	}

	if objectName != "" {
		if err := s3utils.IsValidObjectName(objectName); err != nil {
			return "", err
		}
	}

	// set the action type
	urlVal := make(url.Values)
	if objectName != "" {
		urlVal["action"] = []string{CreateObjectAction}
	} else {
		urlVal["action"] = []string{CreateBucketAction}
	}

	reqMeta := requestMeta{
		bucketName: bucketName,
		objectName: objectName,
		urlValues:  urlVal,
		urlRelPath: "get-approval",
	}

	sendOpt := sendOptions{
		method:     http.MethodGet,
		isAdminApi: true,
	}

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt, authInfo)
	if err != nil {
		log.Printf("get approval rejected: %s \n", err.Error())
		return "", err
	}

	// fetch primary sp signature from sp response
	signature := resp.Header.Get(HTTPHeaderPreSignature)
	if signature == "" {
		return "", errors.New("fail to fetch pre createObject signature")
	}

	return signature, nil
}

// GetPieceHashRoots return primary pieces Hash and secondary piece Hash roots list and object size
// It is used for generate meta of object on the chain
func (c *Client) GetPieceHashRoots(reader io.Reader, segSize int64, ecShards int) (string, []string, int64, error) {
	pieceHashRoots, size, err := SplitAndComputerHash(reader, segSize, ecShards)
	if err != nil {
		log.Println("get hash roots fail", err.Error())
		return "", nil, 0, err
	}

	return pieceHashRoots[0], pieceHashRoots[1:], size, nil
}
