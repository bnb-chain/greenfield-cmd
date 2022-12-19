package inscription

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var emptyURL = url.URL{}

const (
	HTTPHeaderContentLength = "Content-Length"
	HTTPHeaderContentMD5    = "Content-MD5"
	HTTPHeaderContentType   = "Content-Type"
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

	// Validate incoming endpoint URL.
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
		return toInvalidArgumentResp("Endpoint paths not invalid")
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

func calcMD5(body io.Reader) (b64 string) {
	// Small body, use memory
	buf, _ := ioutil.ReadAll(body)
	sum := md5.Sum(buf)
	b64 = base64.StdEncoding.EncodeToString(sum[:])
	return
}

func decodeURIComponent(s string) (string, error) {
	decodeStr, err := url.QueryUnescape(s)
	if err != nil {
		return s, err
	}
	return decodeStr, err
}

// getHostAddr returns host header if available, otherwise returns host from URL
func getHostAddr(req *http.Request) string {
	host := req.Header.Get("host")
	if host != "" && req.Host != host {
		return host
	}
	if req.Host != "" {
		return req.Host
	}
	return req.URL.Host
}

//  addQueryValues adds queryValue to url
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
