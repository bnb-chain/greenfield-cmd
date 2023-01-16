package sign

import (
	"log"
	"os"

	"github.com/bnb-chain/bfs/app/params"
	client "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	signkeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// EIP712Signer is EIP 712 address txn signer
type EIP712Signer struct {
	chainId   string
	clientCtx client.Context
	addr      sdk.AccAddress
	privKey   cryptotypes.PrivKey
	signData  authsigning.SignerData
	keyring   signkeyring.Keyring
	pubKey    cryptotypes.PubKey
}

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
func MakeEncodingConfig() params.EncodingConfig {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, []signing.SignMode{
		signing.SignMode_SIGN_MODE_EIP_712,
	})

	return params.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          txCfg,
	}
}

// initCtx init the client.Context with signMode EIP712 and read the config file to init parameter
func initCtx(homeDir string) (client.Context, error) {
	encodingConfig := MakeEncodingConfig()
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		//	WithKeyringOptions(keyring.ETHAlgoOption()).
		WithInput(os.Stdin).
		WithHomeDir(homeDir).
		WithViper("").WithAccountRetriever(authtypes.AccountRetriever{})

	clientCtx, err := config.ReadFromClientConfig(initClientCtx)
	if err != nil {
		log.Printf("read config to init client fail:%s\n", err.Error())
		return client.Context{}, err
	}
	log.Println("client ctx chain id :" + clientCtx.ChainID)

	return clientCtx, nil
}

// NewSigner create a signer of inscription txns
func NewSigner(address sdk.AccAddress, homeDir string, key cryptotypes.PrivKey, pubKey cryptotypes.PubKey) (*EIP712Signer, error) {
	clientCtx, err := initCtx(homeDir)
	if err != nil {
		log.Printf("init signer fail:%s address: %s \n", err.Error(), address.String())
		return nil, err
	}

	s := &EIP712Signer{
		clientCtx: clientCtx,
		addr:      address,
		chainId:   clientCtx.ChainID,
		privKey:   key,
		keyring:   clientCtx.Keyring,
		pubKey:    pubKey,
	}
	return s, nil
}

func (e *EIP712Signer) GetCtx() client.Context {
	return e.clientCtx
}

func (e *EIP712Signer) GetChainId() string {
	return e.chainId
}

func (e *EIP712Signer) GetSignerData() authsigning.SignerData {
	return e.signData
}

func (s *EIP712Signer) genSignData(pubKey cryptotypes.PubKey, accNum uint64, accSeq uint64) authsigning.SignerData {
	signerdata := authsigning.SignerData{
		Address:       s.addr.String(),
		ChainID:       s.chainId,
		AccountNumber: accNum,
		Sequence:      accSeq,
		PubKey:        pubKey,
	}
	s.signData = signerdata
	return signerdata
}

// SignTxn return the signature of txn
func (s *EIP712Signer) SignTxn(txBuilder client.TxBuilder) (signing.SignatureV2, error) {
	accNum, accSeq, err :=
		s.clientCtx.AccountRetriever.
			GetAccountNumberSequence(s.clientCtx, s.addr)

	signerData := s.genSignData(s.pubKey, accNum, accSeq)

	signMode := signing.SignMode_SIGN_MODE_EIP_712
	signBytes, err := s.clientCtx.TxConfig.SignModeHandler().GetSignBytes(signMode, signerData, txBuilder.GetTx())

	if err != nil {
		log.Printf("get sign bytes fail: %s", err.Error())
		return signing.SignatureV2{}, err
	}

	sigBytes, err := s.privKey.Sign(signBytes)
	if err != nil {
		log.Printf("privateKey sign fail:%s", err.Error())
		return signing.SignatureV2{}, err
	}

	sigBytes[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	var prevSignatures []signing.SignatureV2

	log.Printf("sign txn %s \n", string(sigBytes))
	// Construct the SignatureV2 struct
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: sigBytes,
	}

	sig := signing.SignatureV2{
		PubKey:   s.pubKey,
		Data:     &sigData,
		Sequence: accSeq,
	}

	prevSignatures = append(prevSignatures, sig)

	err = txBuilder.SetSignatures(prevSignatures...)
	if err != nil {
		log.Printf("tx builder set sig err %s", err.Error())
	}

	return sig, nil
}

func (s *EIP712Signer) GetTxnSignBytes(txBuilder client.TxBuilder) ([]byte, error) {
	_, err := s.SignTxn(txBuilder)
	if err != nil {
		log.Printf("sign txn fail: %s, address: %s \n", err.Error(), s.pubKey.String())
		return []byte(""), err
	}

	txBytes, err := s.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		log.Printf("encode txn fail: %s, address: %s \n", err.Error(), s.pubKey.String())
		return []byte(""), err
	}

	/*
		data, err := rlp.EncodeToBytes(txBytes)
		if err != nil {
			return []byte(""), err
		}
	*/
	return txBytes, nil
}
