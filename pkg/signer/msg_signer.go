package signer

import (
	"fmt"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
)

// MsgSigner defines a type that for signing msg in the way that is same with MsgEthereumTx
type MsgSigner struct {
	privKey cryptotypes.PrivKey
}

func NewMsgSigner(sk cryptotypes.PrivKey) *MsgSigner {
	return &MsgSigner{
		privKey: sk,
	}
}

// Sign signs the message using the underlying private key
func (m MsgSigner) Sign(msg []byte) ([]byte, cryptotypes.PubKey, error) {
	if m.privKey.Type() != ethsecp256k1.KeyType {
		return nil, nil, fmt.Errorf(
			"invalid private key type, expected %s, got %s", ethsecp256k1.KeyType, m.privKey.Type(),
		)
	}

	sig, err := m.privKey.Sign(msg)
	if err != nil {
		return nil, nil, err
	}

	return sig, m.privKey.PubKey(), nil
}

// RecoverAddr recover the sender address from msg and signature
func RecoverAddr(msg []byte, sig []byte) (sdk.AccAddress, ethsecp256k1.PubKey, error) {
	pubKeyByte, err := secp256k1.RecoverPubkey(msg, sig)
	if err != nil {
		return nil, ethsecp256k1.PubKey{}, err
	}
	pubKey, _ := ethcrypto.UnmarshalPubkey(pubKeyByte)
	pk := ethsecp256k1.PubKey{
		Key: ethcrypto.CompressPubkey(pubKey),
	}

	recoverAcc := sdk.AccAddress(pk.Address().Bytes())

	return recoverAcc, pk, nil
}
