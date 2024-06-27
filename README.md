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

- **CommitmentManager** [0x9b96...EeF8](https://sepolia.etherscan.io/address/0x9b96A7F97eff734B761bFD9fEBe9928a43E8EeF8)
    - Key Functions:
        - `SubmitCommitment()`: Submit commitment and signature information.
        - `indices()`: Get the current user's commitment index.
- **NodeManager** [0x2B2a...53d5](https://sepolia.etherscan.io/address/0x2B2aa5FAe80433D02619Cfe20348d47DD8E653d5)
    - Key Functions:
        - `RegisterBroadcastNode`: Stake tokens to become a broadcast node.
        - `RegisterStorageNode`: Stake tokens to become a storage node.
- **StorageManager** [0x8B3F...3376](https://sepolia.etherscan.io/address/0x8B3Fd50373219Ff1708a2aB34E77937273463376)
    - Key Functions:
        - `CreateNameSpace`: Create a group of storage nodes for long-term data storage.
        - `StoreAddressMapping`: Create a group of broadcast nodes for signing, and set the required minimum number of signatures.


