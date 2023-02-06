package greenfield

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	ec "github.com/bnb-chain/greenfield-storage-provider/pkg/redundancy"
)

var emptyURL = url.URL{}

const (
	HTTPHeaderContentLength   = "Content-Length"
	HTTPHeaderContentMD5      = "Content-MD5"
	HTTPHeaderContentType     = "Content-Type"
	HTTPHeaderTransactionHash = "X-Gnfd-Txn-Hash"
	HTTPHeaderResource        = "X-Gnfd-Resource"
	HTTPHeaderPreSignature    = "X-Gnfd-Pre-Signature"
	HTTPHeaderDate            = "X-Gnfd-Date"
	HTTPHeaderEtag            = "ETag"
	HTTPHeaderRange           = "Range"
	HTTPHeaderUserAgent       = "User-Agent"
	HTTPHeaderContentSHA256   = "X-Gnfd-Content-Sha256"

	// EmptyStringSHA256 is the hex encoded sha256 value of an empty string
	EmptyStringSHA256       = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
	iso8601DateFormatSecond = "2006-01-02T15:04:05Z"

	AdminURLPrefix  = "/greenfield/admin"
	AdminURLVersion = "/v1"

	CreateObjectAction = "CreateObject"
	CreateBucketAction = "CreateBucket"
	SegmentSize        = 16 * 1024 * 1024
	EncodeShards       = 6
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
		return errors.New("Endpoint url is empty.")
	}

	if endpointURL.Path != "/" && endpointURL.Path != "" {
		return errors.New("Endpoint paths invalid")
	}

	host := endpointURL.Hostname()
	if !CheckIP(host) {
		msg := endpointURL.Host + " does not meet ip address standards."
		return errors.New(msg)
	}

	if !CheckDomainName(host) {
		msg := endpointURL.Host + " does not meet domain name standards."
		return errors.New(msg)
	}

	return nil
}

// CalcSHA256Hex compute checksum of sha256 hash and encode it to hex
func CalcSHA256Hex(buf []byte) (hexStr string) {
	sum := CalcSHA256(buf)
	hexStr = hex.EncodeToString(sum)
	return
}

// CalcSHA256 compute checksum of sha256 from byte array
func CalcSHA256(buf []byte) []byte {
	h := sha256.New()
	h.Write(buf)
	sum := h.Sum(nil)
	return sum[:]
}

// CalcSHA256HashByte compute checksum of sha256 from io.reader
func CalcSHA256HashByte(body io.Reader) ([]byte, error) {
	if body == nil {
		return []byte(""), errors.New("body empty")
	}
	buf := make([]byte, 1024)
	h := sha256.New()
	if _, err := io.CopyBuffer(h, body, buf); err != nil {
		return []byte(""), err
	}
	hash := h.Sum(nil)
	return hash, nil
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

// SplitAndComputerHash split the reader into segment, ec encode the data, compute the hash roots of pieces,
// and return the hash result array list and data size
func SplitAndComputerHash(reader io.Reader, segmentSize int64, ecShards int) ([]string, int64, error) {
	var segChecksumList [][]byte
	var result []string
	encodeData := make([][][]byte, ecShards)
	seg := make([]byte, segmentSize)

	contentLen := int64(0)
	// read the data by segment size
	for {
		n, err := reader.Read(seg)
		if err != nil {
			if err != io.EOF {
				log.Println("content read error:", err)
				return nil, 0, err
			}
			break
		}
		if n > 0 {
			contentLen += int64(n)
			// compute segment hash
			segmentReader := bytes.NewReader(seg[:n])
			if segmentReader != nil {
				checksum, err := CalcSHA256HashByte(segmentReader)
				if err != nil {
					log.Println("compute checksum err:", err)
					return nil, 0, err
				}
				segChecksumList = append(segChecksumList, checksum)
			}

			// get erasure encode bytes
			encodeShards, err := ec.EncodeRawSegment(seg[:n])

			if err != nil {
				log.Println("erasure encode err:", err)
				return nil, 0, err
			}

			for index, shard := range encodeShards {
				encodeData[index] = append(encodeData[index], shard)
			}
		}
	}

	// combine the hash root of pieces of the PrimarySP
	segBytesTotal := bytes.Join(segChecksumList, []byte(""))
	segmentRootHash := CalcSHA256Hex(segBytesTotal)
	result = append(result, segmentRootHash)

	// compute the hash root of pieces of the SecondarySP
	var wg = &sync.WaitGroup{}
	spLen := len(encodeData)
	wg.Add(spLen)
	hashList := make([]string, spLen)
	for spID, content := range encodeData {
		go func(data [][]byte, id int) {
			defer wg.Done()
			var checksumList [][]byte
			for _, pieces := range data {
				piecesHash := CalcSHA256(pieces)
				checksumList = append(checksumList, piecesHash)
			}

			piecesBytesTotal := bytes.Join(checksumList, []byte(""))
			hashList[id] = CalcSHA256Hex(piecesBytesTotal)
		}(content, spID)
	}
	wg.Wait()

	for i := 0; i < spLen; i++ {
		result = append(result, hashList[i])
	}

	return result, contentLen, nil
}
