package signer

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/stretchr/testify/require"
)

func TestSigner(t *testing.T) {
	privKey, _, addr := testdata.KeyEthSecp256k1TestPubAddr()
	rawdata := []byte("this is a test stringToSign content")
	// generate signed string bytes
	stringToSign := crypto.Keccak256(rawdata)
	fmt.Println("str to be signed:", stringToSign)
	signer := NewMsgSigner(privKey)
	signature, orignPubKey, err := signer.Sign(addr.String(), stringToSign)
	require.NoError(t, err)
	fmt.Println("origin pubkey:", orignPubKey.Bytes())
	fmt.Println("origin addr:", addr.String())

	// recover the sender addr
	recoverAcc, pk, err := RecoverAddr(stringToSign, signature)
	require.NoError(t, err)

	fmt.Println("recover pubkey:", recoverAcc.String())
	if !addr.Equals(recoverAcc) {
		t.Errorf("recover addr not same")
	}

	// verify the signature
	verifySucc := secp256k1.VerifySignature(pk.Bytes(), stringToSign, signature[:len(signature)-1])
	if !verifySucc {
		t.Errorf("verify fail")
	}
}

func TestMsgSign(t *testing.T) {
	urlmap := url.Values{}

	// client actions, new request , sign the request
	urlmap.Add("client_id", "test")
	parms := io.NopCloser(strings.NewReader(urlmap.Encode()))
	req, err := http.NewRequest("POST", "www.baidu.com", parms)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = "testHost"

	privKey, _, addr := testdata.KeyEthSecp256k1TestPubAddr()
	req, err = SignRequest(*req, addr, privKey)
	require.NoError(t, err)

	// server action,get the header,verify header and check data
	authHeader := req.Header.Get(HTTPHeaderAuthorization)
	if authHeader == "" {
		t.Errorf("authorization header should not be empty")
	}
	fmt.Println("authorization header:", authHeader)

}
