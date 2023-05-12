package main

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type Key struct {
	Address    sdk.AccAddress
	PrivateKey string // the hex string of the ethsecp256k1 privKey
}

type encryptedKey struct {
	Address string              `json:"address"`
	Crypto  keystore.CryptoJSON `json:"crypto"`
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
	keyBytes := []byte(key.PrivateKey)
	cryptoStruct, err := keystore.EncryptDataV3(keyBytes, []byte(auth), scryptN, scryptP)
	if err != nil {
		return nil, err
	}
	keyJSON := encryptedKey{
		key.Address.String(),
		cryptoStruct,
	}
	return json.Marshal(keyJSON)
}

// DecryptKey decrypts a key from a json blob, returning the private key hex string
func DecryptKey(keyJson []byte, auth string) (string, error) {
	k := new(encryptedKey)
	if err := json.Unmarshal(keyJson, k); err != nil {
		return "", err
	}
	keyBytes, err := decryptKey(k, auth)
	if err != nil {
		return "", err
	}

	return string(keyBytes), nil
}

func decryptKey(key *encryptedKey, auth string) (keyBytes []byte, err error) {
	plainText, err := keystore.DecryptDataV3(key.Crypto, auth)
	if err != nil {
		return nil, err
	}
	return plainText, err
}
