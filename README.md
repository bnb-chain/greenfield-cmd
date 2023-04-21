# greenfield-cmd

---
Greenfield client cmd tool, supporting commands to make requests to greenfield


## Disclaimer
**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Cmd usage

### basic config 

The command should run with "-c filePath" to load the config file and the config should be toml format

Config file example:
```
# the primary storage provider endpoint
endpoint = "https://gnfd-testnet-sp-1.nodereal.io"
# the grpc address of greenfield
grpcAddr = "gnfd-testnet-fullnode-cosmos-us.bnbchain.org:9090"
# the chain id info of greenfield
chainId = "greenfield_5600-1"
privateKey = "ec9577ceafbfa462d510e505df63aba8f8b23886fefxxxxxxxxxxxxx"
```

### support commands

```
COMMANDS:
   make-bucket             create a new bucket
   update-bucket           update bucket meta on chain
   put                     upload payload of object to SP
   get                     download an object
   put-obj-policy          put object policy to group or account
   create-obj              create an object on chain
   cancel                  cancel creating object
   get-hash                compute the integrity hash of file
   del-obj                 delete an existed object
   del-bucket              delete an existed bucket
   head-obj                query object info
   head-bucket             query bucket info
   ls-sp                   list storage providers info
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

```
// get help for supporing commands and basic command format
gnfd-cmd -h

// get help of specific commands
gnfd-cmd command-name -h 
```

### Precautions

1. The private key of the account has to be configured in config file

2. The operator account should have enough balance before sending request to greenfield

### Examples

#### Account Operations
```
// transfer to an account in Greenfield
gnfd-cmd -c config.toml transfer --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// query the balance of account
gnfd-cmd -c config.toml balance --address 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d

// create a payment account
gnfd-cmd -c config.toml create-payment-account

// query payments account under owner or a address with optional flag --user 
gnfd-cmd -c config.toml ls-payment-account --owner 0x5a64aCD8DC6Ce41d824638419319409246A9b41A

// deposit from owner's account to the payment account 
gnfd-cmd -c config.toml payment-deposit --toAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345

// witharaw from a payment account to owner's account
gnfd-cmd -c config.toml payment-withdraw --fromAddress 0xF678C3734F0EcDCC56cDE2df2604AC1f8477D55d --amount 12345
```

#### Bucket Operations

```
// create bucket
gnfd-cmd -c config.toml make-bucket gnfd://bucketname

// update bucket visibility, charged quota or payment address
(1) gnfd-cmd -c config.toml update-bucket  --visibility=public-read  gnfd://cmdbucket78
(2) gnfd-cmd -c config.toml update-bucket  --chargedQuota 50000 gnfd://cmdbucket78
```
#### Upload/Download Operations

(1) first stage of uploading: create a new object on greenfield chain
```
gnfd-cmd -c config.toml  create-obj --contentType "text/xml" --visibility private file-path  gnfd://bucketname/objectname
```
(2) second stage of uploading : upload payload to greenfield storage provide

```
gnfd-cmd -c config.toml put --txnHash xxx  file-path  gnfd://bucketname/objectname
```
required param:  --txnHash

(3) download object

```
gnfd-cmd -c config.toml get gnfd://bucketname/objectname  file-path 
```
#### Group Operations

```
// create group
gnfd-cmd -c config.toml make-group gnfd://groupname

// update group member
gnfd-cmd -c config.toml update-group --addMembers 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// head group member
gnfd-cmd -c config.toml head-member --headMember 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname
```
#### List Operations

```
// list buckets
gnfd-cmd -c config.toml ls-bucket 

// list objects
gnfd-cmd -c config.toml ls gnfd://bucketname

```
#### Delete Operations

```
// delete bucekt
gnfd-cmd -c config.toml del-bucket gnfd://bucketname

//delete object
gnfd-cmd -c config.toml del-obj gnfd://bucketname/objectname

// delete group
gnfd-cmd -c config.toml del-group gnfd://group-name
```
#### Head Operations

```
// head bucekt
gnfd-cmd -c config.toml head-bucket gnfd://bucket-name

// head object
gnfd-cmd -c config.toml head-obj gnfd://bucket-name/object-name

// head Group
gnfd-cmd -c config.toml head-group gnfd://groupname
```

#### Policy Operations
```
// The object policy actions can be "create", “delete”, "copy", "get" or "execute"
// It can be set with combined string like "create,delete" by --actions
// The object policy effect can set to be allow or deny by --effect

// grant object operation permissions to a group
gnfd-cmd -c config.toml put-obj-policy --groupId 128  --actions get,delete  gnfd://group-name/object-name

// grant object operation permissions to an account
gnfd-cmd -c config.toml put-obj-policy --granter 0x169321fC04A12c16...  --actions get,delete gnfd://group-name/object-name

```

#### Storage Provider Operations

```
// list storage providers
gnfd-cmd -c config.toml ls-sp

// get quota price of storage provider:
gnfd-cmd -c config.toml get-price --spAddress 0x70d1983A9A76C8d5d80c4cC13A801dc570890819
```
#### Payment Operations

```
// get quota info
gnfd-cmd -c config.toml quota-info gnfd://bucketname

// buy quota
gnfd-cmd -c config.toml buy-quota --chargedQuota 1000000 gnfd://bucket-name
```
#### Hash Operations

```
// compute integrity hash
gnfd-cmd  -c config.toml get-hash filepath

```

#### Crosschain Operations
```
// crosschain transfer some tokens to an account in BSC
gnfd-cmd -c config.toml transfer-out --toAddress "0x2eDD53b48726a887c98aDAb97e0a8600f855570d" --amount 12345

// mirror a group to BSC
gnfd-cmd -c config.toml mirror --resource group --id 1

// mirror a bucket to BSC
gnfd-cmd -c config.toml mirror --resource bucket --id 1

// mirror a object to BSC
gnfd-cmd -c config.toml mirror --resource object --id 1
```
