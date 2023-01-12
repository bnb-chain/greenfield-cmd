package inscription

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/stretchr/testify/require"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	mux *http.ServeMux

	// client is the COS client being tested.
	client *Client

	// server is a test HTTP server used to provide mock API responses.
	server *httptest.Server
)

// setup sets up a test HTTP server along with  Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	privKey, pubKey, addr := testdata.KeyEthSecp256k1TestPubAddr()

	var err error
	fmt.Println("server url:", server.URL)
	client, err = NewClient(server.URL[len("http://"):], &Options{}, addr, privKey, pubKey)
	if err != nil {
		log.Fatal("create client  fail")
	}
}

func shutdown() {
	server.Close()
}

func startHandle(t *testing.T, r *http.Request) {
	t.Logf("start handle, Request method: %v, ", r.Method)
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header string, want string) {
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %q, want %q", header, got, want)
	}
}

func getUrl(r *http.Request) string {
	return r.URL.String()
}

func testBody(t *testing.T, r *http.Request, want string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Error reading request body: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("request Body is %s, want %s", got, want)
	}
}

func TestNewClient(t *testing.T) {
	mux_temp := http.NewServeMux()
	server_temp := httptest.NewServer(mux_temp)
	privKey, pubKey, addr := testdata.KeyEthSecp256k1TestPubAddr()

	fmt.Println("server url:", server_temp.URL)
	c, err := NewClient(server_temp.URL[7:], &Options{}, addr, privKey, pubKey)
	if err != nil {
		t.Errorf("new client fail %s", err.Error())
	}
	if got, want := c.GetAgent(), UserAgent; got != want {
		t.Errorf("NewClient UserAgent is %v, want %v", got, want)
	}

	bucketName := "testBucket"
	objectName := "testObject"
	want := "http://testBucket." + server_temp.URL[7:] + "/testObject"
	got, _ := c.generateURL(bucketName, objectName, "", nil, false)
	if got.String() != want {
		t.Errorf("URL is %v, want %v", got, want)
	}

}

func TestGetApproval(t *testing.T) {
	setup()
	defer shutdown()

	bucketName := "test-bucket"
	ObjectName := "test-object"
	signature := "test-signature"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startHandle(t, r)
		testMethod(t, r, "GET")
		url := getUrl(r)
		if strings.Contains(url, CreateObjectAction) {
			testHeader(t, r, HTTPHeaderResource, bucketName+"/"+ObjectName)
		} else if strings.Contains(url, CreateBucketAction) {
			testHeader(t, r, HTTPHeaderResource, bucketName)
		}

		w.Header().Set(HTTPHeaderPreSignature, signature)
		w.WriteHeader(200)
	})

	// test preCreateBucket
	gotSign, _, err := client.GetApproval(context.Background(), bucketName, "")
	require.NoError(t, err)

	if gotSign != signature {
		t.Errorf("get signature err")
	}

	//test preCreateObject
	gotSign, _, err = client.GetApproval(context.Background(), bucketName, ObjectName)

	require.NoError(t, err)

	if gotSign != signature {
		t.Errorf("get signature err")
	}

}
