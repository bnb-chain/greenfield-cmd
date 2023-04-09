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
endpoint = "sp.gnfd.cc"
grpcAddr = "gnfd-grpc-plaintext.qa.bnbchain.world:9090"
chainId = "greenfield_9000-1741"
privateKey = "ec9577ceafbfa462d510e505df63aba8f8b23886fefxxxxxxxxxxxxx"
```

### support commands

```
COMMANDS:
   mb             create bucket
   update-bucket  update bucket meta on chain
   put            upload an object
   get            download an object
   create-object  create an object
   get-hash       compute hash roots of object
   del-obj        delete an existed object
   del-bucket     delete an existed bucket
   head-obj       query object info
   head-bucket    query bucket info
   challenge      Send challenge request
   list-sp        list sp info
   mg             create group
   update-group   update group member
   head-group     query group info
   head-member    check group member if it exists
   del-group      delete an existed group
   buy-quota      update bucket meta on chain
   get-price      get the quota price of sp
   quota-info     get quota info of the bucket
   ls-bucket      list bucket info of the provided user
   ls             list object info of the bucket
```

#### Get help

```
// get help for supporing commands and basic command format
gnfd-cmd -h

// For specific commands, use the following command:
gnfd-cmd command-name -h 
```

### Precautions

1.If the private key has not been configured, the tool will generate one and the operator address

2.The operator account should have balance before testing

### Examples

#### Bucket Operations

```
// create bucket
gnfd-cmd -c config.toml mb gnfd://bucketname

// update bucket visibility, charged quota or payment address
(1) gnfd-cmd -c config.toml update-bucket  --visibility=public-read  gnfd://cmdbucket78
(2) gnfd-cmd -c config.toml update-bucket  --chargedQuota 50000 gnfd://cmdbucket78
```

#### Upload/Download Operations

(1) first stage of uploading: create a new object on greenfield chain
```
gnfd-cmd -c config.toml  create-obj --contenType "text/xml" --visibility private file-path  gnfd://bucketname/objectname
```
(2) second stage of uploading : upload payload to greenfield storage provide

```
gnfd-cmd -c config.toml put --txnhash xxx  file-path  gnfd://bucketname/objectname
```
required param:  --txnhash

(3) download object

```
gnfd-cmd -c config.toml get gnfd://bucketname/objectname  file-path 
```

### Group Operations

```
// create group
gnfd-cmd -c config.toml mg gnfd://groupname

// update group member
gnfd-cmd -c config.toml update-group --addMembers 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname

// head group member
gnfd-cmd -c config.toml head-member --headMember 0xca807A58caF20B6a4E3eDa3531788179E5bc816b gnfd://groupname
```

### List Operations

```
// list buckets
gnfd-cmd -c config.toml ls-bucket 

// list objects
gnfd-cmd -c config.toml ls gnfd://bucketname

```

#### Delete Operations

```
// delete bucekt:
gnfd-cmd -c config.toml del-bucket gnfd://bucketname

//delete object:
gnfd-cmd -c config.toml del-obj gnfd://bucketname/objectname
```

#### Head Operations

```
// head bucekt:
gnfd-cmd -c config.toml head-bucket gnfd://bucket-name

// head object:
gnfd-cmd -c config.toml head-obj gnfd://bucket-name/object-name

// head Group
gnfd-cmd -c config.toml head-group gnfd://groupname
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
// get quota info:
gnfd-cmd -c config.toml quota-info gnfd://bucketname

// buy quota:
gnfd-cmd -c config.toml buy-quota --chargedQuota 1000000 gnfd://bucket-name
```

#### Hash Operations

```
// compute integrity hash
gnfd-cmd  -c config.toml get-hash filepath

// get challenge result
gnfd-cmd -c config.toml challenge --objectId "test" --pieceIndex 2  --spIndex -1
```
