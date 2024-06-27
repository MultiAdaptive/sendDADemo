package main

import (
	"github.com/ethereum/go-ethereum/common"
	"log"
	"testing"
)

func TestSendDADemo(t *testing.T) {
	const nodeGroupKeyStr = "0E11D57F4F8CE1CB8651F79DA743480BDA8B82D7D3C3A2DA2F09D43DB411ED08"
	const nameSpaceId = 0
	sendDADemo(nodeGroupKeyStr, nameSpaceId)

}

func TestGetBroadcastingNodes(t *testing.T) {
	list, err := GetBroadcastingNodes()
	if err != nil {
		log.Printf(err.Error())
	}
	for i, info := range list {
		log.Printf("%d Url:%s  Address:%s  Name:%s  Location:%s  StakedTokens:%s  MaxStorageSpace:%s  ", i, info.Url, info.Addr, info.Name, info.Location, info.StakedTokens, info.MaxStorageSpace)
	}
}

func TestGetStorageNodes(t *testing.T) {
	list, err := GetStorageNodes()
	if err != nil {
		log.Printf(err.Error())
	}
	for i, info := range list {
		log.Printf("%d Url:%s  Address:%s  Name:%s  Location:%s  StakedTokens:%s  MaxStorageSpace:%s  ", i, info.Url, info.Addr, info.Name, info.Location, info.StakedTokens, info.MaxStorageSpace)
	}
}

func TestCreateNodeGroup(t *testing.T) {
	var addrs []common.Address
	addrs = append(addrs, common.HexToAddress("0x11C0bd88eC60e1517ACb072f824Ddc8390AA66C0"), common.HexToAddress("0xDFD3b45c915d2C1a4eb3339F4a6aF75bC7A5A1AE"))
	hash := CreateNodeGroup(2, addrs)
	log.Printf("nodeGroupKeys: %s", hash.Hex())
}

func TestCreateNameSpace(t *testing.T) {
	var addrs []common.Address
	addrs = append(addrs, common.HexToAddress("0x321940E1D175E30Aa73FA9448Fa1cA9033a2e3a7"), common.HexToAddress("0x18422BFB8eFA5Fd160005A5F65c401cB35344fA3"))
	id := CreateNameSpace(addrs)
	if id != 0 {
		log.Printf("nameSpaceId: %d", id)
	}
}
