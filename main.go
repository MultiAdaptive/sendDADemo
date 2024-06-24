package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/kzg"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	kzgsdk "github.com/multiAdaptive/kzg-sdk"
	"log"
	"math/big"
	"time"
)

const dataSize = 5 * 1024 * 1024
const privateKey = ""
const senderStr = ""
const nodeGroupKeyStr = "41fcd5661160df12f1de283a786389d769bef07db59292c92663c9d6e994f99f"
const nameSpaceId = 1
const cmManagerAddress = "0x9b96A7F97eff734B761bFD9fEBe9928a43E8EeF8"
const chainID = 11155111
const node01Url = "http://54.151.240.239:8545"
const node02Url = "http://54.80.136.172:8545"
const ethUrl = "https://eth-sepolia.public.blastapi.io"

func main() {
	// init SDK
	sdk, err := kzgsdk.InitMultiAdaptiveSdk("./srs")
	if err != nil {
		println("kzgsdk Error", err.Error())
		return
	}

	data := simulatedData()

	// generate commitments and proofs
	cm, proof, err := sdk.GenerateDataCommitAndProof(data)
	if err != nil {
		println("kzgsdk Error", err.Error())
		return
	}
	length := uint64(len(data))

	nodeGroupKey := common.HexToHash(nodeGroupKeyStr)
	sender := common.HexToAddress(senderStr)
	index, err := getIndex(sender)
	if err != nil {
		println("getIndex Error", err)
		return
	}

	sign1, err := signature(node01Url, sender, index, length, cm.Marshal(), data, nodeGroupKey, proof.H.Marshal(), proof.ClaimedValue.Marshal())
	if err != nil {
		println("sign1 Error", err)
		return
	}
	sign2, err := signature(node02Url, sender, index, length, cm.Marshal(), data, nodeGroupKey, proof.H.Marshal(), proof.ClaimedValue.Marshal())
	if err != nil {
		println("sign2 Error", err)
		return
	}

	SendCommitToL1(length, nodeGroupKey, [][]byte{sign1, sign2}, cm)
}

func simulatedData() []byte {
	data := make([]byte, dataSize)
	rand.Read(data)
	return data
}

func getIndex(sender common.Address) (uint64, error) {
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return uint64(0), err
	}
	contractAddress := common.HexToAddress(cmManagerAddress)
	instance, err := NewCommitmentManager(contractAddress, client)
	if err != nil {
		return uint64(0), err
	}
	index, err := instance.Indices(nil, sender)
	if err != nil {
		return uint64(0), err

	}
	return index.Uint64(), nil
}

func signature(url string, sender common.Address, index, length uint64, commitment, data []byte, nodeGroupKey [32]byte, proof []byte, claimedValue []byte) ([]byte, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var result []byte
	err = client.Client().CallContext(ctx, &result, "eth_sendDAByParams", sender, index, length, commitment, data, nodeGroupKey, proof, claimedValue)
	return result, err
}

func SendCommitToL1(length uint64, dasKey [32]byte, sign [][]byte, commit kzg.Digest) {
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	private, _ := PrivateKeyToAddress(privateKey)
	contractAddress := common.HexToAddress(cmManagerAddress)
	instance, err := NewCommitmentManager(contractAddress, client)
	if err != nil {
		errStr := fmt.Sprintf("cant create contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(private, big.NewInt(chainID)) // For Mainnet
	if err != nil {
		log.Fatal(err)
	}

	commitData := PairingG1Point{
		X: new(big.Int).SetBytes(commit.X.Marshal()),
		Y: new(big.Int).SetBytes(commit.Y.Marshal()),
	}
	tx, err := instance.SubmitCommitment(auth, big.NewInt(int64(length)), dasKey, sign, big.NewInt(nameSpaceId), commitData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("txhash:", tx.Hash().Hex())
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		errStr := fmt.Sprintf("cant WaitMined by contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("fail")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!")
}

func PrivateKeyToAddress(key string) (*ecdsa.PrivateKey, common.Address) {
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Fatal(err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return privateKey, fromAddress
}
