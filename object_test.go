package greenfield

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/bnb-chain/greenfield-sdk-go/pkg/signer"
	"github.com/stretchr/testify/require"
)

func TestPutObject(t *testing.T) {
	setup()
	defer shutdown()

	bucketName := "testbucket"
	ObjectName := "testobject"

	reader := bytes.NewReader([]byte("test content of object"))
	length, err := GetContentLength(reader)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startHandle(t, r)
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Type", contentDefault)
		testHeader(t, r, "Content-Length", strconv.FormatInt(length, 10))
		testBody(t, r, "test content of object")
	})

	txnHash := "test hash"
	newReader := bytes.NewReader([]byte("test content of object"))

	meta := ObjectMeta{
		ObjectSize:  length,
		ContentType: contentDefault,
	}
	_, err = client.PutObject(context.Background(), bucketName,
		ObjectName, newReader, txnHash, meta, signer.NewAuthInfo(false, ""))
	require.NoError(t, err)
}

func TestGetObject(t *testing.T) {
	setup()
	defer shutdown()

	bucketName := "test-bucket"
	ObjectName := "test-object"

	bodyContent := "test content of object"
	etag := "test etag"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startHandle(t, r)
		testMethod(t, r, "GET")

		w.Header().Set("Etag", etag)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		w.Write([]byte(bodyContent))
	})

	body, info, err := client.GetObject(context.Background(), bucketName, ObjectName, GetObjectOptions{}, signer.NewAuthInfo(false, ""))
	require.NoError(t, err)

	buf := new(strings.Builder)
	io.Copy(buf, body)
	// check download content
	if buf.String() != bodyContent {
		t.Errorf("download content not same")
	}
	// check etag
	if info.Etag != etag {
		t.Errorf("etag error")
		fmt.Println("etag", info.Etag)
	}
}
