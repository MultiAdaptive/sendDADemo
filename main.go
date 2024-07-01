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
const privateKey = "e5eae5cc49dade024474874c4a05a93b6c2e1c97cb35bdcb88971d551b200f33"
const cmManagerAddress = "0xd7cC67a93843fCCf90Baa05fe80c71dd66722f3f"
const NodeManagerAddress = "0xAb89CE3AeDE6EdC5167afE7220E9067e048cE1B1"
const StorageManagerAddress = "0x1fb64EA0E454E5E2359084eBf9e06c8684A91f2A"
const chainID = 11155111
const ethUrl = "https://eth-sepolia.public.blastapi.io"

type RPCSignResult struct {
	SigData   []byte `json:"SigData"`
	TimeStamp int64  `json:"TimeStamp"`
}

func main() {

}

func sendDADemo(nodeGroupKeyStr string, nameSpaceId int64) {
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
	_, sender := PrivateKeyToAddress(privateKey)

	index, err := getIndex(sender)
	if err != nil {
		println("getIndex Error", err)
		return
	}

	signatures, timeout, err := GetSignature(nodeGroupKey, sender, index, length, cm.Marshal(), data, proof.H.Marshal(), proof.ClaimedValue.Marshal())
	if err != nil {
		println("GetSignature Error", err)
		return
	}

	SendCommitToL1(length, nodeGroupKey, signatures, cm, nameSpaceId, timeout)
}

func GetSignature(nodeGroupKey common.Hash, sender common.Address, index, length uint64, commitment, data []byte, proof []byte, claimedValue []byte) ([][]byte, int64, error) {
	timeout := int64(0)
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, timeout, err
	}
	storageManagerAddress := common.HexToAddress(StorageManagerAddress)
	storageManager, err := NewStorageManager(storageManagerAddress, client)
	if err != nil {
		return nil, timeout, err
	}

	nodeManagerAddress := common.HexToAddress(NodeManagerAddress)
	nodeManager, err := NewNodeManager(nodeManagerAddress, client)
	if err != nil {
		return nil, timeout, err
	}

	nodeGroup, err := storageManager.NODEGROUP(nil, nodeGroupKey)
	if err != nil {
		return nil, timeout, err
	}
	var re [][]byte
	for _, add := range nodeGroup.Addrs {
		info, err := nodeManager.BroadcastingNodes(nil, add)
		if err != nil {

			re = append(re, nil)
			continue
		}
		sign, err := signature(info.Url, sender, index, length, commitment, data, nodeGroupKey, proof, claimedValue)
		if err != nil {
			re = append(re, nil)
			continue
		}
		re = append(re, sign.SigData)
		timeout = sign.TimeStamp
	}
	return re, timeout, nil
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

func signature(url string, sender common.Address, index, length uint64, commitment, data []byte, nodeGroupKey [32]byte, proof []byte, claimedValue []byte) (*RPCSignResult, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var result RPCSignResult
	err = client.Client().CallContext(ctx, &result, "eth_sendDAByParams", sender, index, length, commitment, data, nodeGroupKey, proof, claimedValue)
	return &result, err
}

func SendCommitToL1(length uint64, dasKey [32]byte, sign [][]byte, commit kzg.Digest, nameSpaceId int64, timeout int64) {
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
	tx, err := instance.SubmitCommitment(auth, big.NewInt(int64(length)), big.NewInt(timeout), big.NewInt(nameSpaceId), dasKey, sign, commitData)
	if err != nil {
		log.Fatal(err)
	}
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		errStr := fmt.Sprintf("cant WaitMined by contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("fail")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!", tx.Hash().Hex())
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

func GetBroadcastingNodes() ([]NodeManagerNodeInfo, error) {
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, err
	}
	contractAddress := common.HexToAddress(NodeManagerAddress)
	instance, err := NewNodeManager(contractAddress, client)
	if err != nil {
		return nil, err
	}
	nodeList, err := instance.GetBroadcastingNodes(nil)
	if err != nil {
		return nil, err

	}
	return nodeList, nil
}

func GetStorageNodes() ([]NodeManagerNodeInfo, error) {
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, err
	}
	contractAddress := common.HexToAddress(NodeManagerAddress)
	instance, err := NewNodeManager(contractAddress, client)
	if err != nil {
		return nil, err
	}
	nodeList, err := instance.GetBroadcastingNodes(nil)
	if err != nil {
		return nil, err

	}
	return nodeList, nil
}

func CreateNodeGroup(requiredAmountOfSignatures int64, addrs []common.Address) common.Hash {
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	private, _ := PrivateKeyToAddress(privateKey)
	contractAddress := common.HexToAddress(StorageManagerAddress)
	instance, err := NewStorageManager(contractAddress, client)
	if err != nil {
		errStr := fmt.Sprintf("cant create contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(private, big.NewInt(chainID)) // For Mainnet
	if err != nil {
		log.Fatal(err)
	}

	tx, err := instance.StoreAddressMapping(auth, big.NewInt(requiredAmountOfSignatures), addrs)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("txhash:", tx.Hash().Hex())
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		errStr := fmt.Sprintf("cant WaitMined by contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("fail")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!", tx.Hash().Hex())
	return receipt.Logs[0].Topics[2]
}

func CreateNameSpace(addrs []common.Address) uint64 {
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	private, _ := PrivateKeyToAddress(privateKey)
	contractAddress := common.HexToAddress(StorageManagerAddress)
	instance, err := NewStorageManager(contractAddress, client)
	if err != nil {
		errStr := fmt.Sprintf("cant create contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(private, big.NewInt(chainID)) // For Mainnet
	if err != nil {
		log.Fatal(err)
	}

	tx, err := instance.CreateNameSpace(auth, addrs)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("txhash:", tx.Hash().Hex())
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		errStr := fmt.Sprintf("cant WaitMined by contract address err:%s", err.Error())
		log.Fatal(errStr)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("fail")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!", tx.Hash().Hex())
	for _, l := range receipt.Logs {
		if info, err := instance.ParseNameSpaceCreated(*l); err == nil {
			return info.Id.Uint64()
		}
	}
	return 0
}
