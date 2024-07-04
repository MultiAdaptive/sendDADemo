# Send DA Demo


# Prerequisites

- View all broadcasting nodes  
    ```shell
      go test -run TestGetBroadcastingNodes
    ```
- View all storage nodes  
    ```shell
      go test -run TestGetStorageNodes
    ```
- Register nodeGroup  

  Register the broadcasting node group and set the minimum number of signatures required. Their signatures will be needed when uploading commitments.
    ```shell
      go test -run TestCreateNodeGroup
    ```
- Register nameSpace   

  This step is optional. If long-term storage is needed, specify the storage nodes and complete the registration.  
  ```shell
    go test -run TestCreateNameSpace
  ```

## Flow
1. Select your desired broadcasting nodes to register a nodeGroup and set the required minimum number of signatures.
2. If long-term storage is needed, you also need to register a nameSpace to specify the long-term storage nodes.
3. Pass in the SRS file path to initialize the kzg-sdk.
4. Use the SDK's GenerateDataCommitAndProof method with the DA data to generate commitments and proof.
5. Send the DA data, commitments, proof, and other information to the broadcast node to request a signature.
6. Send the obtained signature, commitments, and other information to the contract.(If long-term storage is not needed, please set `nameSpaceId` to 0.)

## Terminology

- **Commitment**: A cryptographic construct used to commit to a value while keeping it hidden until later.
- **Proof**: A piece of evidence or a demonstration that something is true or valid.
- **nodeGroup**: Group multiple broadcast nodes together and set the required minimum number of signatures.
- **nameSpace**: Group multiple storage nodes together to designate them for long-term storage.
- **sender**: Address of the user sending the DA data.
- **index**: Data index stored under the user's address.
- **signature**: Sign the specified data and broadcast the node's signature to indicate receipt and temporary storage of DA data.


## Contract

- **CommitmentManager** [0xa8ED...A724](https://sepolia.etherscan.io/address/0xa8ED91Eb2B65A681A742011798d7FB31C50FA724)
    - Key Functions:
        - `SubmitCommitment()`: Submit commitment and signature information.
        - `indices()`: Get the current user's commitment index.
- **NodeManager** [0x97bE...67EC](https://sepolia.etherscan.io/address/0x97bE3172AEA87b60224e8d604aC4bAbe55F067EC)
    - Key Functions:
        - `RegisterBroadcastNode`: Stake tokens to become a broadcast node.
        - `RegisterStorageNode`: Stake tokens to become a storage node.
- **StorageManager** [0x6642...30fe](https://sepolia.etherscan.io/address/0x664250Fb3b1cd58f07683D957A34daf8A06130fe)
    - Key Functions:
        - `CreateNameSpace`: Create a group of storage nodes for long-term data storage.
        - `StoreAddressMapping`: Create a group of broadcast nodes for signing, and set the required minimum number of signatures.


