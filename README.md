# greenfield-cmd

---
Greenfield client cmd tool, supporting commands to make requests to greenfield


## Disclaimer
**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Cmd usage

### installation

```
git clone https://github.com/bnb-chain/greenfield-cmd.git
cd greenfield-cmd
make build
cd build
./gnfd-cmd -h
```

### basic config 

The command tool supports the "--home" option to specify the path of the config file and the keystore, the default path is a directory called ".gnfd-cmd" under the home directory of the system.
When running commands that interact with the greenfield, if there is no config/config.toml file under the path and the commands runs without "--config" flag, 
the tool will generate the config/config.toml file automatically which is consistent with the testnet configuration under the path.

Below is an example of the config file. The rpcAddr and chainId should be consistent with the Greenfield network.
For Greenfield Testnet, you can refer to [Greenfield Testnet RPC Endpoints](https://greenfield.bnbchain.org/docs/guide/resources.html#rpc-endpoints).
The rpcAddr indicates the Tendermint RPC address with the port info.
```
rpcAddr = "https://gnfd-testnet-fullnode-tendermint-us.bnbchain.org:443"
chainId = "greenfield_5600-1"
```
The command tool can also support other networks besides the test network. 
you can replace the content of a custom config file in the default config directory with config.toml or
run command with "-c filepath" to set the custom config file.


#### Get help

The commands support different kinds of commands, including bucket,object,group,bank,policy,sp,payment-account and keystore.
```
// get help for supporing commands and basic command format
gnfd-cmd -h
   bucket           support the bucket operation functions, including create/update/delete/head/list and so on
   object           support the object operation functions, including put/get/update/delete/head/list and so on
   group            support the group operation functions, including create/update/delete/head/head-member/mirror
   bank             support the bank functions, including transfer in greenfield and query balance
   policy           support object,bucket and group policy operation functions
   payment-account  support the payment account operation functions
   sp               support the storage provider operation functions
   keystore         support the keystore operation functions

```

The following command can be used to obtain help information for commands. For example, you can use "gnfd-cmd object -h" to obtain the subcommand infos under the object command.
```
gnfd-cmd [command-name] -h
```

The following command can be used to obtain help information for subcommands. For example, you can use "gnfd-cmd object update -h" to obtain the help info to update object.
```
gnfd-cmd [command-name][subcommand-name] -h
```

### Precautions

1. The user need to use "keystore create" command to generate a keystore file first. The content of the keystore is the encrypted private key information.
All the other commands need run with -k if the keystore is not on the default path.

2. The operator account should have enough balance before sending request to greenfield.

3. The cmd tool has ability to intelligently select the correct SP by the info of bucket name and object name in command. Users do not need to specify the address of the SP in the command or config file.

4. The "gnfd://" is a fixed prefix which representing the greenfield object or bucket.


### Examples

#### Generate Keystore

Before generate keystore, you should export your private key from MetaMask and write it into a local file as plaintext.

Users can use "keystore generate" to generate the key file with the flag "--privKeyFile" which indicates the private key plaintext file .
The keystore will be generated in the path "keystore/key.json" under the home directory of the system or the directory set by "-home".
Password info is also needed to run the command. The terminal will prompt user to enter the password information. After the terminal obtains user's password information,
the password file will store in the path "keystore/password/password.txt" under the home directory of the system or the directory set by "-home".
Users can also specify the password file path by using the "--passwordfile".

```
// generate keystore key.json, key.txt contain the plaintext private key
// After the keystore file has been generated, user can delete the private key file key.txt.
gnfd-cmd keystore generate --privKeyFile key.txt 
```

Users can use "keystore inspect" to display the keystore information include publicKey, address and privateKey.
```
// display the keystore info
gnfd-cmd keystore inspect --privateKey true
```

#### Account Operations
```
// transfer to an account in Greenfield
gnfd-cmd bank transfer --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// crosschain transfer some tokens to an account in BSC
gnfd-cmd bank bridge --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// query the balance of account
gnfd-cmd bank balance --address 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d

```

#### Storage Provider Operations

The "sp" command is used to list the SP and query the information of SP.
```
// list storage providers
gnfd-cmd sp ls

// get storage provider info
gnfd-cmd sp head https://gnfd-testnet-sp-1.nodereal.io

// get quota and storage price of storage provider:
gnfd-cmd sp get-price https://gnfd-testnet-sp-1.nodereal.io
```

#### Bucket Operations

Before creating bucket, it is recommended to first run the "sp ls" command to obtain the SP list information of Greenfield,
and then select the target SP to which the bucket will be created on.

```
// create bucket. 
// The targt primary SP address to which the bucket will be created on need to be set by --primarySP flag.
// If the primary SP has not been not set, the cmd will choose first SP in the SP list which obtain from chain as the primary SP.
gnfd-cmd bucket create gnfd://gnfd-bucket

// update bucket visibility, charged quota or payment address
(1) gnfd-cmd bucket update --visibility=public-read gnfd://gnfd-bucket
(2) gnfd-cmd bucket update --chargedQuota 50000 gnfd://gnfd-bucket
```
#### Upload/Download Operations

(1) put Object

The "object put" command is used to upload a file from local which is less than 2G. The bucket name and object name should be replaced with specific names and
the file-path should replace by the file path of local system.
```
gnfd-cmd object put --contentType "text/xml" --visibility private file-path gnfd://gnfd-bucket/gnfd-object
```
if the object name has not been set, the command will use the file name as object name. If you need upload a file to the folder, you need to run "object put" command with "--folder" flag.

The tool also support create a folder on bucket by "object create-folder" command.
```
gnfd-cmd object create-folder gnfd://gnfd-bucket/testfolder
```

(2) download object

The "object get" command is used to download an object to local path. This command will return the local file path where the object will be downloaded and the file size after successful execution.
```
gnfd-cmd object get gnfd://gnfd-bucket/gnfd-object file-path 
```
The filepath can be a specific file path, a directory path, or not set at all. 
If not set, the command will download the content to a file with the same name as the object name in the current directory. If it is set as a directory, the command will download the object file into the directory.

#### Group Operations

The group commands is used to create group, update group members, delete group and query group info.
```
// create group
gnfd-cmd group create gnfd://groupname

// update group member
gnfd-cmd group update --addMembers 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// head group member
gnfd-cmd group head-member  0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// delete group
gnfd-cmd group delete gnfd://group-name
```
#### Policy Operations

The gnfd-cmd policy command supports the policy for put/delete resources policy(including objects, buckets, and groups) to the principal.

The principal is need to be set by --grantee which indicates a greenfield account or --groupId which indicates group id.

The object policy action can be "create", "delete", "copy", "get" , "execute", "list" or "all".
The bucket policy actions can be "update", "delete", "create", "list", "update", "getObj", "createObj" and so on.
The group policy actions can be "update", "delete" or all, update indicates the update-group-member action.

The policy effect can set to be "allow" or "deny" by --effect

Put policy examples:
```
// grant object operation permissions to a group
gnfd-cmd policy put  --groupId 128  --actions get,delete  grn:o::gnfd-bucket/gnfd-object

// grant object operation permissions to an account
gnfd-cmd policy put --grantee 0x169321fC04A12c16...  --actions get,delete grn:o::gnfd-bucket/gnfd-object

// grant bucket operation permissions to a group
gnfd-cmd policy put --groupId 130 --actions delete,update,createObj  grn:b::gnfd-bucket

// grant bucket operation permissions to an account
gnfd-cmd policy put --grantee 0x169321fC04A12c16...  --actions delete,update  grn:b::gnfd-bucket

// grant group operation permissions to an account 
gnfd-cmd policy put --grantee 0x169321fC04A12c16...  --actions update  grn:g:owneraddress:gnfd-group
```
Delete policy examples:
```
// delete the bucket policy from an grantee
gnfd-cmd policy delete --grantee   grn:b::gnfd-bucket

// delete the object policy from an grantee
gnfd-cmd policy delete --grantee 0x169321fC04A12c16...  grn:o::gnfd-bucket/gnfd-object

```
#### List Operations
```
// list buckets
gnfd-cmd bucket ls

// list objects of the bucket
gnfd-cmd object ls gnfd://gnfd-bucket

```
#### Delete Operations
```
// delete bucekt
gnfd-cmd bucket delete gnfd://gnfd-bucket

//delete object
gnfd-cmd object delete gnfd://gnfd-bucket/gnfd-object

```
#### Head Operations
```
// head bucekt
gnfd-cmd bucket head gnfd://gnfd-bucket

// head object
gnfd-cmd object head gnfd://gnfd-bucket/gnfd-object

// head Group
gnfd-cmd group head gnfd://groupname
```
#### Payment Operations
```
// create a payment account
gnfd-cmd payment-account create

// list payment accounts under owner or a address with optional flag --user 
gnfd-cmd payment-account ls --owner 0x5a64aCD8DC6Ce41d824638419319409246A9b41A

// deposit from owner's account to the payment account 
gnfd-cmd payment-account deposit --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// witharaw from a payment account to owner's account
gnfd-cmd payment-account withdraw --fromAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345
```

#### Quota Operations
```
// get quota info
gnfd-cmd bucket get-quota gnfd://gnfd-bucket

// buy quota
gnfd-cmd bucket buy-quota --chargedQuota 1000000 gnfd://gnfd-bucket
```

#### Resource mirror Operations

```
// mirror a group as NFT to BSC
gnfd-cmd group mirror  --id 1

// mirror a bucket as NFT to BSC,
gnfd-cmd bucket mirror --id 1

```

## Reference

- [Greenfield](https://github.com/bnb-chain/greenfield): the greenfield blockchain
- [Greenfield-Contract](https://github.com/bnb-chain/greenfield-contracts): the cross chain contract for Greenfield that deployed on BSC network. .
- [Greenfield-Storage-Provider](https://github.com/bnb-chain/greenfield-storage-provider): the storage service infrastructures provided by either organizations or individuals.
- [Greenfield-Relayer](https://github.com/bnb-chain/greenfield-relayer): the service that relay cross chain package to both chains.