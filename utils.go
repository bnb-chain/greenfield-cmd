package inscription

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var emptyURL = url.URL{}

const (
	HTTPHeaderContentLength    = "Content-Length"
	HTTPHeaderContentMD5       = "Content-MD5"
	HTTPHeaderContentType      = "Content-Type"
	HTTPHeadeGnfdContentLength = "X-Gnfd-Content-Length"
	HTTPHeaderTransactionMsg   = "X-Gnfd-Txn-Msg"
	HTTPHeaderTransactionHash  = "X-Gnfd-Txn-Hash"
	HTTPHeaderTransactionDate  = "X-Gnfd-Txn-Date"
	HTTPHeaderResource         = "X-Gnfd-Resource"
	HTTPHeaderPreSignature     = "X-Gnfd-Pre-Signature"
	HTTPHeaderDate             = "X-Gnfd-Date"
	HTTPHeaderEtag             = "ETag"
	HTTPHeaderHost             = "Host"
	HTTPHeaderRange            = "Range"
	HTTPHeaderUserAgent        = "User-Agent"
	HTTPHeaderContentSHA256    = "X-Gnfd-Content-Sha256"

	// Hex encoded string of nil sha256sum bytes.
	emptySHA256Hex = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	// EmptyStringSHA256 is the hex encoded sha256 value of an empty string
	EmptyStringSHA256 = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

	AdminURLPrefix  = "/greenfield/admin"
	AdminURLVersion = "/v1"

	CreateObjectAction = "CreateObject"
	CreateBucketAction = "CreateBucket"
)

func CheckIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// CheckDomainName CheckdDomainName validates if input string is a valid domain name.
func CheckDomainName(hostName string) bool {
	// See RFC 1035, RFC 3696.
	hostName = strings.TrimSpace(hostName)
	if len(hostName) == 0 || len(hostName) > 255 {
		return false
	}
	if hostName[len(hostName)-1:] == "-" || hostName[:1] == "-" {
		return false
	}
	if hostName[len(hostName)-1:] == "_" || hostName[:1] == "_" {
		return false
	}
	if hostName[:1] == "." {
		return false
	}

	if strings.ContainsAny(hostName, "`~!@#$%^&*()+={}[]|\\\"';:><?/") {
		return false
	}
	return true
}

// getEndpointURL - construct a new endpoint.
func getEndpointURL(endpoint string, secure bool) (*url.URL, error) {
	// If secure is false, use 'http' scheme.
	scheme := "https"
	if !secure {
		scheme = "http"
	}

	// Construct a secured endpoint URL.
	endpointURLStr := scheme + "://" + endpoint
	endpointURL, err := url.Parse(endpointURLStr)
	if err != nil {
		return nil, err
	}
	// check endpoint if it is valid
	if err := isValidEndpointURL(*endpointURL); err != nil {
		return nil, err
	}
	return endpointURL, nil
}

// Verify if input endpoint URL is valid.
func isValidEndpointURL(endpointURL url.URL) error {
	if endpointURL == emptyURL {
		return toInvalidArgumentResp("Endpoint url is empty.")
	}

	if endpointURL.Path != "/" && endpointURL.Path != "" {
		return toInvalidArgumentResp("Endpoint paths invalid")
	}

	host := endpointURL.Hostname()
	if !CheckIP(host) {
		msg := endpointURL.Host + " does not meet ip address standards."
		return toInvalidArgumentResp(msg)
	}

	if !CheckDomainName(host) {
		msg := endpointURL.Host + " does not meet domain name standards."
		return toInvalidArgumentResp(msg)
	}

	return nil
}

func calcMD5OfBody(body io.Reader) (b64 string) {
	if body == nil {
		return ""
	}
	buf, _ := io.ReadAll(body)
	m := md5.New()
	m.Write(buf)
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

func calMD5Digest(msg []byte) []byte {
	// TODO: chunk compute
	m := md5.New()
	m.Write(msg)
	return m.Sum(nil)
}

func calcSHA256Hex(buf []byte) (hexStr string) {
	h := sha256.New()
	h.Write(buf)
	sum := h.Sum(nil)
	hexStr = hex.EncodeToString(sum[:])
	return
}

func CalcSHA256Hash(body io.Reader) (b64 string) {
	if body == nil {
		return emptySHA256Hex
	}
	buf, _ := io.ReadAll(body)
	return calcSHA256Hex(buf)
}

func decodeURIComponent(s string) (string, error) {
	decodeStr, err := url.QueryUnescape(s)
	if err != nil {
		return s, err
	}
	return decodeStr, err
}

// addQueryValues adds queryValue to url
func addQueryValues(s string, qs url.Values) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	q := u.RawQuery
	rq := qs.Encode()
	if q != "" {
		if rq != "" {
			u.RawQuery = fmt.Sprintf("%s&%s", q, qs.Encode())
		}
	} else {
		u.RawQuery = rq
	}
	return u.String(), nil
}

// closeResponse close the response body
func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

// getHostInfo returns host header from the request
func GetHostInfo(req http.Request) string {
	host := req.Header.Get(HTTPHeaderHost)
	if host != "" && req.Host != host {
		return host
	}
	if req.Host != "" {
		return req.Host
	}
	return req.URL.Host
}

// GetContentLength return the size of reader
func GetContentLength(reader io.Reader) (int64, error) {
	var contentLength int64
	var err error
	switch v := reader.(type) {
	case *bytes.Buffer:
		contentLength = int64(v.Len())
	case *bytes.Reader:
		contentLength = int64(v.Len())
	case *strings.Reader:
		contentLength = int64(v.Len())
	case *os.File:
		fInfo, fError := v.Stat()
		if fError != nil {
			err = fmt.Errorf("can't get reader content length,%s", fError.Error())
		} else {
			contentLength = fInfo.Size()
		}
	default:
		err = fmt.Errorf("can't get reader content length,unkown reader type")
	}
	return contentLength, err
}
