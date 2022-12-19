package inscription

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
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
	BucketName string
	ObjectName string
}

// Error returns the error msg
func (r ErrResponse) Error() string {
	RequestID := r.RequestID
	if RequestID == "" {
		RequestID = r.Response.Header.Get("X-Amz-Request-Id")
	}

	decodeURL, err := decodeURIComponent(r.Response.Request.URL.String())
	if err != nil {
		decodeURL = r.Response.Request.URL.String()
	}

	var statusCode int
	if r.StatusCode > 0 {
		statusCode = r.StatusCode
	} else {
		statusCode = r.Response.StatusCode
	}
	return fmt.Sprintf("%v %v: %d %v(Message: %v, RequestId: %v)",
		r.Response.Request.Method, decodeURL,
		statusCode, r.Code, r.Message, RequestID)
}

// IsErrorResp check the response is a error response
func IsErrorResp(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &ErrResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		xml.Unmarshal(data, errorResponse)
	}

	return err
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
