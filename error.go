package greenfield

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

/* **** SAMPLE ERROR RESPONSE ****
<?xml version="1.0" encoding="UTF-8"?>
<Error>
   <Code>AccessDenied</Code>
   <Message>Access Denied</Message>
   <RequestId>xxx</RequestId>
   <HostId>xx</HostId>
</Error>
*/

type ErrResponse struct {
	XMLName    xml.Name       `xml:"Error"`
	Response   *http.Response `xml:"-"`
	Code       string
	StatusCode int `xml:"-"`
	Message    string
	Resource   string
	RequestID  string
	Server     string
	BucketName string
	ObjectName string
}

// Error returns the error msg
func (r ErrResponse) Error() string {
	decodeURL := ""
	method := ""
	if r.Response != nil {
		decodeURL, _ = decodeURIComponent(r.Response.Request.URL.String())
		method = r.Response.Request.Method
	}

	return fmt.Sprintf("%v %v: %d %v(Message: %v)",
		method, decodeURL,
		r.StatusCode, r.Code, r.Message)
}

// IsErrorResp check the response is an error response
func construtErrResponse(r *http.Response, bucketName, objectName string) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	if r == nil {
		msg := "Response is empty. "
		return toInvalidArgumentResp(msg)
	}
	// todo(leo) change this
	errorResp := ErrResponse{
		StatusCode: r.StatusCode,
		Server:     r.Header.Get("Server"),
	}

	if errorResp.RequestID == "" {
		errorResp.RequestID = r.Header.Get("X-Gnfd-Request-Id")
	}

	var data []byte
	var readErr error
	if r.Body != nil {
		data, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			errorResp = ErrResponse{
				StatusCode: r.StatusCode,
				Code:       r.Status,
				Message:    readErr.Error(),
				BucketName: bucketName,
			}
		}
	}

	var decodeErr error
	if readErr == nil && data != nil {
		decodeErr = xml.Unmarshal(data, &errorResp)
		if decodeErr != nil {
			log.Printf("unmarshal xml body fail %s", decodeErr)
		}
	}

	if decodeErr != nil || data == nil || r.Body == nil {
		errBody := bytes.TrimSpace(data)

		switch r.StatusCode {
		case http.StatusNotFound:
			if objectName == "" {
				errorResp = ErrResponse{
					StatusCode: r.StatusCode,
					Code:       "NoSuchBucket",
					Message:    "The specified bucket does not exist.",
					BucketName: bucketName,
				}
			} else {
				errorResp = ErrResponse{
					StatusCode: r.StatusCode,
					Code:       "NoSuchObject",
					Message:    "The specified object does not exist.",
					BucketName: bucketName,
					ObjectName: objectName,
				}
			}
		case http.StatusForbidden:
			errorResp = ErrResponse{
				StatusCode: r.StatusCode,
				Code:       "AccessDenied",
				Message:    "no permission to access the resource",
				BucketName: bucketName,
				ObjectName: objectName,
			}
		default:
			msg := "unknown error"
			if len(errBody) > 0 {
				msg = string(errBody)
			}
			errorResp = ErrResponse{
				StatusCode: r.StatusCode,
				Code:       r.Status,
				Message:    msg,
				BucketName: bucketName,
			}
		}
	}

	return errorResp
}

func CheckNotFoundError(e error) bool {
	if e == nil {
		return false
	}
	err, ok := e.(*ErrResponse)
	if !ok {
		return false
	}
	if err.Response != nil && err.Response.StatusCode == 404 {
		return true
	}
	return false
}

// toInvalidArgumentResp - Invalid argument response.
func toInvalidArgumentResp(message string) error {
	return ErrResponse{
		StatusCode: http.StatusBadRequest,
		Code:       "InvalidArgument",
		Message:    message,
		RequestID:  "insription",
	}
}

func objectSizeInvaild(message string) error {
	return ErrResponse{
		StatusCode: http.StatusBadRequest,
		Code:       "InvalidArgument",
		Message:    message,
		RequestID:  "insription",
	}
}

func fieldEmptyResp(message string) error {
	return ErrResponse{
		StatusCode: http.StatusBadRequest,
		Code:       "InvalidArgument",
		Message:    message,
		RequestID:  "insription",
	}
}
