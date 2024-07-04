package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/kzg"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	kzgsdk "github.com/multiAdaptive/kzg-sdk"
)

const (
	dataSize              = 5 * 1024 * 1024                              // Size of the data to be processed
	privateKey            = ""                                           // Private key for signing transactions
	cmManagerAddress      = "0xa8ED91Eb2B65A681A742011798d7FB31C50FA724" // Commitment Manager contract address
	nodeManagerAddress    = "0x97bE3172AEA87b60224e8d604aC4bAbe55F067EC" // Node Manager contract address
	storageManagerAddress = "0x664250Fb3b1cd58f07683D957A34daf8A06130fe" // Storage Manager contract address
	chainID               = 11155111                                     // Chain ID for the Ethereum network
	ethUrl                = "https://eth-sepolia.public.blastapi.io"     // Ethereum node URL
)

func main() {

}

func sendDADemo(nodeGroupKeyStr string, nameSpaceKey [32]byte) {
	// Initialize the KZG SDK
	sdk, err := kzgsdk.InitMultiAdaptiveSdk("./srs")
	if err != nil {
		log.Fatalf("kzgsdk Error: %s", err)
	}

	// Generate simulated data
	data := simulatedData()

	// Generate data commitment and proof
	cm, proof, err := sdk.GenerateDataCommitAndProof(data)
	if err != nil {
		log.Fatalf("kzgsdk Error: %s", err)
	}

	// Convert node group key from string to hash
	nodeGroupKey := common.HexToHash(nodeGroupKeyStr)
	_, sender := PrivateKeyToAddress(privateKey)

	// Get the index of the sender
	index, err := getIndex(sender)
	if err != nil {
		log.Fatalf("getIndex Error: %s", err)
	}

	ti := time.Now()
	timeout := ti.Add(10 * time.Hour).Unix()

	log.Print(len(data))
	// Get signatures from the node group
	signatures, err := GetSignature(nodeGroupKey, sender, index, uint64(len(data)), cm.Marshal(), data, proof.H.Marshal(), proof.ClaimedValue.Marshal(), uint64(timeout))
	if err != nil {
		log.Fatalf("GetSignature Error: %s", err)
	}

	// Send the commitment to Layer 1
	SendCommitToL1(uint64(len(data)), nodeGroupKey, signatures, cm, nameSpaceKey, timeout)
}

func GetSignature(nodeGroupKey common.Hash, sender common.Address, index, length uint64, commitment, data, proof, claimedValue []byte, timeout uint64) (signatures [][]byte, err error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, err
	}

	// Instantiate the Storage Manager contract
	storageManager, err := NewStorageManager(common.HexToAddress(storageManagerAddress), client)
	if err != nil {
		return nil, err
	}

	// Instantiate the Node Manager contract
	nodeManager, err := NewNodeManager(common.HexToAddress(nodeManagerAddress), client)
	if err != nil {
		return nil, err
	}

	// Get the node group information
	nodeGroup, err := storageManager.NODEGROUP(nil, nodeGroupKey)
	if err != nil {
		return nil, err
	}

	// Iterate over each node in the group to get signatures
	for _, add := range nodeGroup.Addrs {
		info, err := nodeManager.BroadcastingNodes(nil, add)
		if err != nil {
			signatures = append(signatures, nil)
			continue
		}
		sign, err := signature(info.Url, sender, index, length, commitment, data, nodeGroupKey, proof, claimedValue, timeout)
		if err != nil {
			signatures = append(signatures, nil)
			continue
		}
		signatures = append(signatures, sign)
	}

	log.Print(len(signatures))
	for _, bytes := range signatures {
		log.Println(common.Bytes2Hex(bytes))
	}
	return signatures, nil
}

func simulatedData() []byte {
	// Generate simulated data of the specified size
	data := make([]byte, dataSize)
	rand.Read(data)
	return data
}

func getIndex(sender common.Address) (uint64, error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return 0, err
	}
	// Instantiate the Commitment Manager contract
	instance, err := NewCommitmentManager(common.HexToAddress(cmManagerAddress), client)
	if err != nil {
		return 0, err
	}
	// Get the index of the sender
	index, err := instance.Indices(nil, sender)
	if err != nil {
		return 0, err
	}
	return index.Uint64(), nil
}

func signature(url string, sender common.Address, index, length uint64, commitment, data []byte, nodeGroupKey [32]byte, proof, claimedValue []byte, timeout uint64) ([]byte, error) {
	// Connect to the Ethereum client at the specified URL
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var result []byte
	// Call the eth_sendDAByParams method to get the signature
	err = client.Client().CallContext(ctx, &result, "eth_sendDAByParams", sender, index, length, commitment, data, nodeGroupKey, proof, claimedValue, timeout)
	return result, err
}

func SendCommitToL1(length uint64, dasKey [32]byte, sign [][]byte, commit kzg.Digest, nameSpaceId [32]byte, timeout int64) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal(err)
	}
	private, _ := PrivateKeyToAddress(privateKey)
	// Instantiate the Commitment Manager contract
	instance, err := NewCommitmentManager(common.HexToAddress(cmManagerAddress), client)
	if err != nil {
		log.Fatalf("cant create contract address err: %s", err)
	}
	// Create a transactor with the private key and chain ID
	auth, err := bind.NewKeyedTransactorWithChainID(private, big.NewInt(chainID))
	if err != nil {
		log.Fatal(err)
	}

	// Prepare the commitment data for the transaction
	commitData := PairingG1Point{
		X: new(big.Int).SetBytes(commit.X.Marshal()),
		Y: new(big.Int).SetBytes(commit.Y.Marshal()),
	}
	// Submit the commitment to the contract
	tx, err := instance.SubmitCommitment(auth, big.NewInt(int64(length)), big.NewInt(timeout), nameSpaceId, dasKey, sign, commitData)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("cant WaitMined by contract address err: %s", err)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("Transaction failed")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!", tx.Hash().Hex())
}

func PrivateKeyToAddress(key string) (*ecdsa.PrivateKey, common.Address) {
	// Convert the private key from hex string to ECDSA
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Fatal(err)
	}
	// Extract the public key and convert it to address
	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	return privateKey, crypto.PubkeyToAddress(*publicKeyECDSA)
}

func GetBroadcastingNodes() ([]NodeManagerNodeInfo, error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, err
	}
	// Instantiate the Node Manager contract
	instance, err := NewNodeManager(common.HexToAddress(nodeManagerAddress), client)
	if err != nil {
		return nil, err
	}
	// Get the list of broadcasting nodes
	nodeList, err := instance.GetBroadcastingNodes(nil)
	if err != nil {
		return nil, err
	}
	var filtered []NodeManagerNodeInfo
	// Filter nodes that have staked tokens
	for _, info := range nodeList {
		if info.StakedTokens.Cmp(big.NewInt(0)) != 0 {
			filtered = append(filtered, info)
		}
	}
	return filtered, nil
}

func GetStorageNodes() ([]NodeManagerNodeInfo, error) {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, err
	}
	// Instantiate the Node Manager contract
	instance, err := NewNodeManager(common.HexToAddress(nodeManagerAddress), client)
	if err != nil {
		return nil, err
	}
	// Get the list of storage nodes
	nodeList, err := instance.GetStorageNodes(nil)
	if err != nil {
		return nil, err
	}
	var filtered []NodeManagerNodeInfo
	// Filter nodes that have staked tokens
	for _, info := range nodeList {
		if info.StakedTokens.Cmp(big.NewInt(0)) != 0 {
			filtered = append(filtered, info)
		}
	}
	return filtered, nil
}

func CreateNodeGroup(requiredAmountOfSignatures int64, addrs []common.Address) common.Hash {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal(err)
	}
	private, _ := PrivateKeyToAddress(privateKey)
	// Instantiate the Storage Manager contract
	instance, err := NewStorageManager(common.HexToAddress(storageManagerAddress), client)
	if err != nil {
		log.Fatalf("cant create contract address err: %s", err)
	}
	// Create a transactor with the private key and chain ID
	auth, err := bind.NewKeyedTransactorWithChainID(private, big.NewInt(chainID))
	if err != nil {
		log.Fatal(err)
	}

	// Submit the address mapping to create a new node group
	tx, err := instance.RegisterNodeGroup(auth, big.NewInt(requiredAmountOfSignatures), addrs)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("cant WaitMined by contract address err: %s", err)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("Transaction failed")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!", tx.Hash().Hex())
	// Return the node group key
	return receipt.Logs[0].Topics[2]
}

func CreateNameSpace(addrs []common.Address) [32]byte {
	// Connect to the Ethereum client
	client, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal(err)
	}
	private, _ := PrivateKeyToAddress(privateKey)
	// Instantiate the Storage Manager contract
	instance, err := NewStorageManager(common.HexToAddress(storageManagerAddress), client)
	if err != nil {
		log.Fatalf("cant create contract address err: %s", err)
	}
	// Create a transactor with the private key and chain ID
	auth, err := bind.NewKeyedTransactorWithChainID(private, big.NewInt(chainID))
	if err != nil {
		log.Fatal(err)
	}

	// Create a new namespace
	tx, err := instance.RegisterNameSpace(auth, addrs)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("cant WaitMined by contract address err: %s", err)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		log.Fatal("Transaction failed")
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "success!", tx.Hash().Hex())
	// Return the namespace ID
	for _, l := range receipt.Logs {
		if info, err := instance.ParseNameSpaceRegistered(*l); err == nil {
			return info.Key
		}
	}
	return common.HexToHash("0x00")
}
