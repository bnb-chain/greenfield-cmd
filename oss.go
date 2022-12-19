package inscription

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"inscription-sdk/pkg/s3utils"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Client implements Amazon S3 compatible methods.
type Client struct {
	// Parsed endpoint url provided by the user.
	endpoint *url.URL
	// Needs allocation.
	client    *http.Client
	userAgent string

	common service

	Bucket *BucketService
	Object *ObjectService

	conf *Config

	// trace related
	isTraceEnabled  bool
	traceErrorsOnly bool
	traceOutput     io.Writer
}

type Config struct {
	RequestBodyClose bool
	Secure           bool // use https or not
	Transport        http.RoundTripper
	RetryOpt         RetryOptions
	UploadLimitSpeed uint64
}

type Options struct {
	// Transport http.RoundTripper
	secure bool
}

type RetryOptions struct {
	Count      int
	Interval   time.Duration
	StatusCode []int
}

type service struct {
	client *Client
}

// Global constants.
const (
	libName = "inscription-go-sdk"
	Version = "v0.0.1"
)

const (
	UserAgent      = "Inscription (" + runtime.GOOS + "; " + runtime.GOARCH + ") " + libName + "/" + Version
	contentTypeXML = "application/xml"
	contentDefault = "application/octet-stream"
)

// GetURL returns the URL of the S3 endpoint.
func (c *Client) GetURL() *url.URL {
	endpoint := *c.endpoint
	return &endpoint
}

// New creates a new client
func NewClient(endpoint string, opts *Options) (*Client, error) {
	url, err := getEndpointURL(endpoint, opts.secure)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	c := &Client{
		client:    httpClient,
		userAgent: UserAgent,
		endpoint:  url,
		conf: &Config{
			RequestBodyClose: false,
			RetryOpt: RetryOptions{
				Count:    3,
				Interval: time.Duration(0),
			},
		},
	}
	c.common.client = c
	c.Bucket = (*BucketService)(&c.common)
	c.Object = (*ObjectService)(&c.common)
	c.Group = (*GroupService)(&c.common)

	return c, nil
}

// requestMeta - contain the values to make a request.
type requestMeta struct {
	bucketName  string
	objectName  string
	queryValues url.Values

	contentType      string
	contentLength    int64
	contentMD5Base64 string // carries base64 encoded md5sum
	contentSHA256Hex string // carries hex encoded sha256sum
}

// getURL make a new target url.
func (c *Client) getURL(bucketName string, objectName string, queryValues url.Values) (*url.URL, error) {
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

	urlStr := scheme + "://" + host + "/"

	if bucketName == "" {
		err := errors.New("no bucketName in path")
		return nil, err
	}
	urlStr = scheme + "://" + bucketName + "." + host + "/"
	if objectName != "" {
		urlStr += s3utils.EncodePath(objectName)
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

func (c *Client) newRequest(ctx context.Context,
	method string, meta requestMeta, body interface{}) (req *http.Request, err error) {

	// construct the url
	desURL, err := c.getURL(meta.bucketName, meta.objectName, meta.queryValues)
	if err != nil {
		return nil, err
	}

	var reader io.Reader

	contentType := ""
	if body != nil {
		if ObjectReader, ok := body.(io.Reader); ok {
			reader = ObjectReader
			if meta.contentType == " " {
				contentType = contentDefault
			}
		} else {
			b, err := xml.Marshal(body)
			if err != nil {
				return nil, err
			}
			contentType = contentTypeXML
			reader = bytes.NewReader(b)
		}
	}

	c.handleBody(req, reader)
	// Initialize a new HTTP request for the method.
	req, err = http.NewRequestWithContext(ctx, method, desURL.String(), reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)

	// contentType is set by user
	if meta.contentType != " " {
		req.Header.Set("Content-Type", meta.contentType)
	} else {
		req.Header.Set("Content-Type", contentType)
	}

	req.Host = req.URL.Host

	if c.conf.RequestBodyClose {
		req.Close = true
	}

	return
}
func (c *Client) doAPI(ctx context.Context, req *http.Request, closeBody bool) (*http.Response, error) {
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
			// Close the body to let the Transport reuse the connection
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	if resp == nil {
		msg := "Response is empty. "
		return nil, toInvalidArgumentResp(msg)
	}

	err = IsErrorResp(resp)
	if err != nil {
		// StatusCode != 2xx when Get Object
		if !c.conf.RequestBodyClose {
			resp.Body.Close()
		}
		// even though there was an error, we still return the response
		// throw the error response to caller
		return resp, err
	}

	return resp, nil
}

// handleBody handles request body
func (c *Client) handleBody(req *http.Request, body io.Reader) {
	reader := body

	readerLen, err := GetContentLength(reader)
	if err == nil {
		req.ContentLength = readerLen
	}
	req.Header.Set(HTTPHeaderContentLength, strconv.FormatInt(req.ContentLength, 10))

	// MD5
	if body != nil && req.Header.Get(HTTPHeaderContentMD5) == "" {
		req.Header.Set(HTTPHeaderContentMD5, calcMD5(body))
	}

	// HTTP body
	rc, ok := reader.(io.ReadCloser)
	if !ok && reader != nil {
		rc = ioutil.NopCloser(reader)
	}

	req.Body = rc
}

// makeReq new restful request, send the message and handle the response
func (c *Client) makeReq(ctx context.Context, method string,
	metadata requestMeta, body interface{}) (res *http.Response, err error) {
	req, err := c.newRequest(ctx, method, metadata, body)
	if err != nil {
		log.Fatalf("new request error: %s , stop send request\n", err.Error())
		return
	}

	resp, err := c.doAPI(ctx, req, c.conf.RequestBodyClose)
	if err != nil {
		log.Fatalf("do api request fail: %s \n", err.Error())
		return
	}
	return resp, nil
}
