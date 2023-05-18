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

The command should run with "-c filePath" to load the config file and the config should be TOML format.
The default config file is "config.toml".

Below is an example of the config file. The rpcAddr and chainId should be consistent with the Greenfield network.
For Greenfield Testnet, you can refer to [Greenfield Testnet RPC Endpoints](https://greenfield.bnbchain.org/docs/guide/resources.html#rpc-endpoints).
The rpcAddr indicates the Tendermint RPC address with the port info.
The configuration for passwordFile is the path to the file containing the password required to generate the keystore.

```
rpcAddr = "https://gnfd-testnet-fullnode-tendermint-us.bnbchain.org:443"
chainId = "greenfield_5600-1"
passwordFile = "password.txt"
```

### support commands

```
COMMANDS:
   make-bucket             create a new bucket
   update-bucket           update bucket meta on chain
   put                     create object on chain and upload payload of object to SP
   get                     download an object
   put-obj-policy          put object policy to group or account
   put-bucket-policy       put bucket policy to group or account
   cancel-create-obj       cancel the created object
   get-hash                compute the integrity hash of file
   del-obj                 delete an existed object
   del-bucket              delete an existed bucket
   head-obj                query object info
   head-bucket             query bucket info
   ls-sp                   list storage providers info
   head-sp                 get storage provider details
   make-group              create a new group
   update-group            update group member
   head-group              query group info
   head-member             check if a group member exists
   del-group               delete an existed group
   buy-quota               update bucket quota info
   get-price               get the quota price of the SP
   quota-info              get quota info of the bucket
   ls-bucket               list buckets of the user
   ls                      list objects of the bucket
   transfer                transfer from your account to a dest account
   transfer-out            transfer from greenfield to a BSC account
   create-payment-account  create a payment account
   payment-deposit         deposit into stream(payment) account
   payment-withdraw        withdraw from stream(payment) account
   ls-payment-account      list payment accounts of the owner
   balance                 query a account's balance
   mirror                  mirror resource to bsc
```



#### Get help

The commands support different categories, including storage,group,bridge,bank,permission and payment 
```
// get help for supporing commands and basic command format
gnfd-cmd -h
   storage     support the storage functions, including create/put/get/list resource
   group       support the group operation functions
   bridge      support the bridge functions, including transfer and mirror
   bank        support the bank functions, including transfer and get balance
   permission  support object policy and bucket policy operation functions
   payment     support the payment operation functions
   ls-sp       list storage providers info
   head-sp     get storage provider details
   gen-key     generate new keystore file
```

The following command can be used to obtain help information for classified commands. For example, you can use "gnfd-cmd storage -h" to obtain the subcommand infos under the storage command.
```
gnfd-cmd [category-name] -h
```

The following command can be used to obtain help information for subcommands. For example, you can use "gnfd-cmd storage make-bucket -h" to obtain the help info of "make-bucket".
```
gnfd-cmd [category-name][command-name] -h
```
### Precautions

1. The user need to use "gen-key" command to generate a keystore file first. The content of the keystore is the encrypted private key information, 
and the passwordFile is used for encrypting/decrypting the private key. The other commands need run with -k if the keystore is not the default file path(key.json).

2. The operator account should have enough balance before sending request to greenfield.

3. The cmd tool has ability to intelligently select the correct SP by the info of bucket name and object name in command. Users do not need to specify the address of the SP in the command or config file.

4. The "gnfd://" is a fixed prefix which representing the greenfield resources


### Examples

#### Generate Keystore

Before generate keystore, you should export your private key from MetaMask and write it into a local file as plaintext .
You need also write your password on the password file which set by the "passwordFile" field in the config file.

Assuming that the current private key hex string is written as plaintext in the file key.txt，
the following command can be used to generate a keystore file called key.json:
```
// generate keystore key.json
gnfd-cmd gen-key --privKeyFile key.txt  key.json
```

After the keystore file has been generated, you can delete the private key file which contains the plaintext of private key.

#### Account Operations
```
// transfer to an account in Greenfield
gnfd-cmd bank transfer --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// query the balance of account
gnfd-cmd bank balance --address 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d

// create a payment account
gnfd-cmd payment create-payment-account

// query payments account under owner or a address with optional flag --user 
gnfd-cmd payment ls-payment-account --owner 0x5a64aCD8DC6Ce41d824638419319409246A9b41A

// deposit from owner's account to the payment account 
gnfd-cmd payment  payment-deposit --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// witharaw from a payment account to owner's account
gnfd-cmd payment  payment-withdraw --fromAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345
```

#### Storage Provider Operations

THis command is used to list the SP and query the information of SP.
```
// list storage providers
gnfd-cmd ls-sp

// get storage provider info
./gnfd-cmd head-sp --spEndpoint https://gnfd-testnet-sp-1.nodereal.io
```

#### Bucket Operations

Before creating bucket, It is recommended to first run the "ls-sp" command to obtain the SP list information of Greenfield,
and then select the target SP to which the bucket will be created on.

```
// create bucket. 
// The primary SP address which the bucket will be created at need to be set by --primarySP
// If the primary SP has not been not set, the cmd will choose SP0 in the SP list as the primary sp
gnfd-cmd storage make-bucket --primarySP  gnfd://gnfd-bucket

// update bucket visibility, charged quota or payment address
(1) gnfd-cmd storage update-bucket  --visibility=public-read  gnfd://gnfd-bucket
(2) gnfd-cmd storage update-bucket  --chargedQuota 50000 gnfd://gnfd-bucket
```
#### Upload/Download Operations


(1) put Object

The "storage put" command is used to upload a file from local which is less than 2G. The bucket name and object name should be replaced with specific names and
the file-path should replace by the file path of local system.
```
gnfd-cmd storage put --contentType "text/xml" --visibility private file-path  gnfd://gnfd-bucket/gnfd-object
```

The tool also support create a folder on bucket by "storage make-folder" command.
```
./gnfd-cmd storage make-folder  gnfd://gnfd-bucket/test-folder
```

If you need upload a file to the folder , you need to run "storage put" command with "-folder" flag

(2) download object

The "storage get" command is used to download an object to local path, the file-path should replace by the file path of local system.
```
gnfd-cmd storage get gnfd://gnfd-bucket/gnfd-object  file-path 
```


#### Group Operations

The group commands is used to create group, update group members, delete group and query group info.
```
// create group
gnfd-cmd group make-group gnfd://groupname

// update group member
gnfd-cmd group update-group --addMembers 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// head group member
gnfd-cmd group head-member --headMember 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// delete group
gnfd-cmd storage del-group gnfd://group-name
```

#### Permission  Operations
```
// The object policy actions can be "create", “delete”, "copy", "get" or "execute"
// The bucket policy actions can be "update" or "delete"， "update" indicate the updating bucket info permission
// The actions info can be set with combined string like "create,delete" by --actions
// The policy effect can set to be allow or deny by --effect

// grant object operation permissions to a group
gnfd-cmd permission put-obj-policy --groupId 128  --actions get,delete  gnfd://gnfd-bucket/gnfd-object

// grant object operation permissions to an account
gnfd-cmd permission put-obj-policy --grantee 0x169321fC04A12c16...  --actions get,delete gnfd://gnfd-bucket/gnfd-object

// grant bucket operation permissions to a group
gnfd-cmd permission put-bucket-policy --groupId 130 --actions delete,update  gnfd://gnfd-bucket

// grant bucket operation permissions to an account
gnfd-cmd permission put-bucket-policy  --grantee 0x169321fC04A12c16...  --actions delete,update  gnfd://gnfd-bucket

```

#### List Operations

```
// list buckets
gnfd-cmd storage ls-bucket 

// list objects
gnfd-cmd storage ls gnfd://gnfd-bucket

```
#### Delete Operations

```
// delete bucekt
gnfd-cmd storage del-bucket gnfd://gnfd-bucket

//delete object
gnfd-cmd storage del-obj gnfd://gnfd-bucket/gnfd-object

```
#### Head Operations

```
// head bucekt
gnfd-cmd storage head-bucket gnfd://gnfd-bucket

// head object
gnfd-cmd storage head-obj gnfd://gnfd-bucket/gnfd-object

// head Group
gnfd-cmd group head-group gnfd://groupname
```


#### Payment Operations

```
// get quota info
gnfd-cmd payment quota-info gnfd://gnfd-bucket

// buy quota
gnfd-cmd payment buy-quota --chargedQuota 1000000 gnfd://gnfd-bucket

// get quota price of storage provider:
gnfd-cmd payment get-price --spAddress 0x70d1983A9A76C8d5d80c4cC13A801dc570890819
```
#### Hash Operations

```
// compute integrity hash
gnfd-cmd storage get-hash filepath

```

#### Crosschain Operations
```
// crosschain transfer some tokens to an account in BSC
gnfd-cmd crosschain transfer-out --toAddress "0x2eDD53b48726a887c98aDAb97e0a8600f855570d" --amount 12345

// mirror a group to BSC
gnfd-cmd crosschain mirror --resource group --id 1

// mirror a bucket to BSC
gnfd-cmd crosschain mirror --resource bucket --id 1

// mirror a object to BSC
gnfd-cmd crosschain mirror --resource object --id 1
```
