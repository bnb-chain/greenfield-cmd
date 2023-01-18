package signer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"

	"github.com/bnb-chain/greenfield-sdk-go/pkg/s3utils"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	HTTPHeaderAuthorization   = "Authorization"
	signAlgorithm             = "ECDSA-secp256k1"
	HTTPHeaderTransactionDate = "X-Gnfd-Txn-Date"
)

// getCanonicalHeaders generate a list of request headers with their values
func getCanonicalHeaders(req http.Request) string {
	var content bytes.Buffer
	var containHostHeader bool
	for header, value := range req.Header {
		if header == "Authorization" || header == HTTPHeaderTransactionDate {
			continue
		}
		content.WriteString(strings.ToLower(header))
		content.WriteByte(':')

		if header != "host" {
			for i, v := range value {
				if i > 0 {
					content.WriteByte(',')
				}
				trimVal := strings.Join(strings.Fields(v), " ")
				content.WriteString(trimVal)
			}
			content.WriteByte('\n')
		} else {
			containHostHeader = true
			content.WriteString(GetHostInfo(req))
			content.WriteByte('\n')
		}
	}

	if !containHostHeader {
		content.WriteString(GetHostInfo(req))
		content.WriteByte('\n')
	}

	return content.String()
}

// getSignedHeaders return the alphabetically sorted, semicolon-separated list of lowercase request header names.
func getSignedHeaders(req http.Request) string {
	var signHeaders []string
	for k := range req.Header {
		headerKey := http.CanonicalHeaderKey(k)
		if headerKey != "Authorization" && headerKey != "User-Agent" {
			signHeaders = append(signHeaders, strings.ToLower(k))
		}
	}
	sort.Strings(signHeaders)
	return strings.Join(signHeaders, ";")
}

// getCanonicalRequest generate the canonicalRequest base on aws s3 sign without payload hash.
// https://docs.aws.amazon.com/general/latest/gr/create-signed-request.html#create-canonical-request
func getCanonicalRequest(req http.Request) string {
	req.URL.RawQuery = strings.ReplaceAll(req.URL.Query().Encode(), "+", "%20")
	canonicalRequest := strings.Join([]string{
		req.Method,
		s3utils.EncodePath(req.URL.Path),
		req.URL.RawQuery,
		getCanonicalHeaders(req),
		getSignedHeaders(req),
	}, "\n")

	return canonicalRequest
}

// GetStringToSign generate the string from canonicalRequest to sign
func GetStringToSign(req http.Request) string {
	time := req.Header.Get(HTTPHeaderTransactionDate)
	canonicalRequest := getCanonicalRequest(req)
	stringToSign := time + hex.EncodeToString(calcSHA256([]byte(canonicalRequest)))

	return stringToSign
}

// SignRequest sign the request before send to server
// http://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html.
func SignRequest(req http.Request, addr sdk.AccAddress, privKey cryptotypes.PrivKey) (*http.Request, error) {
	stringToSign := GetStringToSign(req)
	// sign the request header info, generate the signature
	signer := NewMsgSigner(privKey)
	signature, _, err := signer.Sign(addr.String(), crypto.Keccak256([]byte(stringToSign)))
	if err != nil {
		return &req, err
	}

	authStr := []string{
		signAlgorithm + "StringToSign=" + stringToSign,
		"Signature=" + hex.EncodeToString(signature),
	}

	// set auth header
	req.Header.Set(HTTPHeaderAuthorization, strings.Join(authStr, ", "))

	return &req, nil
}

func calcSHA256(msg []byte) (sum []byte) {
	h := sha256.New()
	h.Write(msg)
	sum = h.Sum(nil)
	return
}

// getHostInfo returns host header from the request
func GetHostInfo(req http.Request) string {
	host := req.Header.Get("host")
	if host != "" && req.Host != host {
		return host
	}
	if req.Host != "" {
		return req.Host
	}
	return req.URL.Host
}
