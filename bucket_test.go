package inscription

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateBucket(t *testing.T) {
	setup()
	defer shutdown()

	bucketName := "testbucket"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startHandle(t, r)
		testMethod(t, r, "PUT")
	})

	err := client.CreateBucket(context.Background(), bucketName, false)
	if err != nil {
		t.Fatalf("Bucket.Put returned error: %v", err)
	}

}
