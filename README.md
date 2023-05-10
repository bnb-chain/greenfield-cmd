# greenfield-cmd

---
Greenfield client cmd tool, supporting commands to make requests to greenfield


## Disclaimer
**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Cmd usage

### basic config 

The command should run with "-c filePath" to load the config file and the config should be toml format.
The default config file is "config.toml"

Config file example:
```
rpcAddr = "gnfd-testnet-fullnode-cosmos-us.bnbchain.org:9090"
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

the commands support different categories, including storage,group,bridge,bank,permission and payment 
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
   gen-key     generate new keystore file
```

get help of specific category commands
```
gnfd-cmd [category-name] -h

for example : gnfd-cmd stroage -h
```

get help of specific commands 
```
gnfd-cmd [category-name][command-name] -h

for example : gnfd-cmd stroage create-bucket -h
```
### Precautions

1. The private key of the account has to be configured in config file

2. The operator account should have enough balance before sending request to greenfield

3. The cmd tool has ability to intelligently select the correct SP

4. The "gnfd://" is a fixed prefix which representing the greenfield resources

5. gnfd-cmd need run with --keystore if the keystore is not the default file path

### Examples

#### Generate Keystore
```
// generate keystore key.json
gnfd-cmd gen-key --privKeyFile key.txt --password password.txt  key.json
```

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

#### Bucket Operations

```
// create bucket. 
// The primary SP address which the bucket will be created at need to be set by --primarySP
gnfd-cmd storage create-bucekt --primarySP  gnfd://bucketname


// update bucket visibility, charged quota or payment address
(1) gnfd-cmd storage update-bucket  --visibility=public-read  gnfd://cmdbucket78
(2) gnfd-cmd storage update-bucket  --chargedQuota 50000 gnfd://cmdbucket78
```
#### Upload/Download Operations

(1) put Object
```
gnfd-cmd storage put --contentType "text/xml" --visibility private file-path  gnfd://bucketname/objectname

```
the file-path should replace by the file path of local system

(2) download object

```
gnfd-cmd storage get gnfd://bucketname/objectname  file-path 
```

the file-path should replace by the file path of local system

#### Group Operations

```
// create group
gnfd-cmd group make-group gnfd://groupname

// update group member
gnfd-cmd group update-group --addMembers 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// head group member
gnfd-cmd group head-member --headMember 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname
```
#### List Operations

```
// list buckets
gnfd-cmd storage ls-bucket 

// list objects
gnfd-cmd storage ls gnfd://bucketname

```
#### Delete Operations

```
// delete bucekt
gnfd-cmd storage del-bucket gnfd://bucketname

//delete object
gnfd-cmd storage del-obj gnfd://bucketname/objectname

// delete group
gnfd-cmd storage del-group gnfd://group-name
```
#### Head Operations

```
// head bucekt
gnfd-cmd storage head-bucket gnfd://bucket-name

// head object
gnfd-cmd storage head-obj gnfd://bucket-name/object-name

// head Group
gnfd-cmd group head-group gnfd://groupname
```

#### Permission  Operations
```
// The object policy actions can be "create", “delete”, "copy", "get" or "execute"
// The bucket policy actions can be "update" or "delete"， "update" indicate the updating bucket info permission
// The actions info can be set with combined string like "create,delete" by --actions
// The policy effect can set to be allow or deny by --effect

// grant object operation permissions to a group
gnfd-cmd permission put-obj-policy --groupId 128  --actions get,delete  gnfd://bucket-name/object-name

// grant object operation permissions to an account
gnfd-cmd permission put-obj-policy --granter 0x169321fC04A12c16...  --actions get,delete gnfd://bucket-name/object-name

// grant bucket operation permissions to a group
gnfd-cmd permission put-bucket-policy --groupId 130 --actions delete,update  gnfd://bucket-name

// grant bucket operation permissions to an account
gnfd-cmd permission put-bucket-policy  --granter 0x169321fC04A12c16...  --actions delete,update  gnfd://bucket-name

```

#### Storage Provider Operations

```
// list storage providers
gnfd-cmd -c config.toml ls-sp

// get quota price of storage provider:
gnfd-cmd payment get-price --spAddress 0x70d1983A9A76C8d5d80c4cC13A801dc570890819
```
#### Payment Operations

```
// get quota info
gnfd-cmd payment quota-info gnfd://bucketname

// buy quota
gnfd-cmd payment buy-quota --chargedQuota 1000000 gnfd://bucket-name
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
