package greenfield

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bnb-chain/greenfield-sdk-go/sign"
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

	conf    *Config
	sender  sdk.AccAddress // sender address
	privKey cryptotypes.PrivKey
	signer  *sign.EIP712Signer
}

type Config struct {
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

// Global constants.
const (
	libName        = "inscription-go-sdk"
	Version        = "v0.0.1"
	UserAgent      = "Inscription (" + runtime.GOOS + "; " + runtime.GOARCH + ") " + libName + "/" + Version
	contentTypeXML = "application/xml"
	contentDefault = "application/octet-stream"
)

// NewClient returns a new greenfield client
func NewClient(endpoint string, opts *Options, addr sdk.AccAddress,
	privKey cryptotypes.PrivKey, pubKey cryptotypes.PubKey) (*Client, error) {
	url, err := getEndpointURL(endpoint, opts.secure)
	if err != nil {
		log.Println("get url error:", err.Error())
		return nil, err
	}
	log.Println("new client with url:", url.String())

	eip712signer, err := sign.NewSigner(addr, ".gnfd", privKey, pubKey)

	httpClient := &http.Client{}
	c := &Client{
		client:    httpClient,
		userAgent: UserAgent,
		endpoint:  url,
		conf: &Config{
			RetryOpt: RetryOptions{
				Count:    3,
				Interval: time.Duration(0),
			},
		},
		sender:  addr,
		privKey: privKey,
		signer:  eip712signer,
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
	bucketName    string
	objectName    string
	urlRelPath    string     // relative path of url
	urlValues     url.Values // url values to be added into url
	Range         string
	ApproveAction string

	contentType       string
	contentLength     int64
	gnfdContentLength int64
	contentMD5Base64  string // base64 encoded md5sum
	contentSHA256     string // hex encoded sha256sum
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
	method string, meta requestMeta, body interface{}, txnHash string, isAdminAPi bool) (req *http.Request, err error) {
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
			sha256Hex = calcSHA256Hex(content)
		}
	}

	// Initialize a new HTTP request for the method.
	req, err = http.NewRequestWithContext(ctx, method, desURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// need to turn the body into ReadCloser
	if meta.contentLength == 0 {
		req.Body = nil
	} else {
		if body != nil {
			req.Body = io.NopCloser(reader)
		}
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
	}

	// set md5 header
	if meta.contentMD5Base64 != "" {
		req.Header[HTTPHeaderContentMD5] = []string{meta.contentMD5Base64}
	}

	// set first stage upload x-gnfd-content-length header
	if meta.gnfdContentLength > 0 {
		req.Header.Set(HTTPHeadeGnfdContentLength, strconv.FormatInt(meta.gnfdContentLength, 10))
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

	// TODO(leo) parse host when sp domain supported
	// set Host from url or client
	if meta.bucketName != "" {
		req.Host = meta.bucketName + ".gnfd.nodereal.com"
	}

	if isAdminAPi {
		req.Host = "gnfd.nodereal.com"
		if meta.bucketName != "" {
			if meta.objectName == "" {
				req.Header.Set(HTTPHeaderResource, meta.bucketName)
			} else {
				req.Header.Set(HTTPHeaderResource, meta.bucketName+"/"+meta.objectName)
			}
		}
	}

	// set date header
	stNow := time.Now().UTC()
	req.Header.Set(HTTPHeaderDate, stNow.Format(http.TimeFormat))

	// set user-agent
	req.Header.Set(HTTPHeaderUserAgent, c.userAgent)

	// sign the total http request info
	if bytes.Compare(c.sender, []byte("")) != 0 && c.privKey != nil {
		req, err = signer.SignRequest(*req, c.sender, c.privKey)
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
func (c *Client) sendReq(ctx context.Context, metadata requestMeta, opt *sendOptions) (res *http.Response, err error) {
	req, err := c.newRequest(ctx, opt.method, metadata, opt.body, opt.txnHash, opt.isAdminApi)
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
		// TODO(leo) temp change, this need be change back after domain supported
		// urlStr := scheme + "://" + bucketName + "." + host + "/"
		urlStr = scheme + "://" + host + "/"
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

// ApproveInfo define the info of the storage provider approval reply
type ApproveInfo struct {
	Resource       string
	Action         string
	SpAddr         sdk.AccAddress
	ExpirationTime time.Time
}

// GetApproval return the signature info for the approval of preCreating resources
func (c *Client) GetApproval(ctx context.Context, bucketName, objectName string) (string, ApproveInfo, error) {
	if err := s3utils.IsValidBucketName(bucketName); err != nil {
		return "", ApproveInfo{}, err
	}

	if objectName != "" {
		if err := s3utils.IsValidObjectName(objectName); err != nil {
			return "", ApproveInfo{}, err
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

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt)
	if err != nil {
		log.Printf("pre create object rejected: %s \n", err.Error())
		return "", ApproveInfo{}, err
	}

	// TODO(leo) need check if need to decode the signature and get ApproveInfo
	signature := resp.Header.Get(HTTPHeaderPreSignature)
	if signature == "" {
		return "", ApproveInfo{}, fmt.Errorf("fail to fetch pre createObject signature")
	}

	return signature, ApproveInfo{}, nil
}

// GetPieceHashRoots return primary pieces Hash and secondary piece Hash
// The first return value is the primary SP piece hash root, the second is the secondary SP piece hash roots list
func (c *Client) GetPieceHashRoots(reader io.Reader, segSize int64) (string, []string, error) {
	pieceHashRoots, err := SplitAndComputerHash(reader, segSize)

	if err != nil {
		return "", nil, err
	}

	return pieceHashRoots[0], pieceHashRoots[1:], nil
}
