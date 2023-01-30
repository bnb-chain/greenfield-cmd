package greenfield

import (
	"context"
	"net/http"
	"testing"
)

// TestCreateBucket test creating a new bucket
func TestCreateBucket(t *testing.T) {
	setup()
	defer shutdown()

	bucketName := "testbucket"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startHandle(t, r)
		testMethod(t, r, "PUT")
		testHeader(t, r, HTTPHeaderContentSHA256, EmptyStringSHA256)
		w.WriteHeader(200)
	})

	err := client.CreateBucket(context.Background(), bucketName, false)
	if err != nil {
		t.Fatalf("Bucket.Put returned error: %v", err)
	}

}
