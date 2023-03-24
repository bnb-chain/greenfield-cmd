# greenfield-cmd

---
Greenfield client cmd tool, supporting commands to make requests to greenfield


## Disclaimer
**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Cmd usage

### basic config 

 config file example
```
endpoint = "http://127.0.0.1:8888"
host = "nodereal.gnfd.com"
grpcAddr = "127.0.0.1:26750"
chainId = "greenfield_9000-1741"
privateKey = "ec9577ceafbfa462d510e505df63aba8f8b23886fefbbda4xxxxxxxx"
```

### support commands

```
COMMANDS:
   mb           create bucket
   put          upload an object
   get          download an object
   create-obj   create an object
   get-hash     compute hash roots of object
   del-obj      delete an existed object
   del-bucket   delete an existed bucket
   head-obj     query object info
   head-bucket  query bucket info
   challenge    Send challenge request
   list-sp      list sp info
```
### Precautions

1.If the private key has not been configured, the tool will generate one and the operator address

2.The operator account should have balance before testing

### Examples

#### List Storage Provider 
```
gnfd-cmd  --config=config.toml list-sp
```

#### Create Bucket

 create bucket: create a new bucket on greenfield chain
```
gnfd-cmd --config=config.toml mb  gnfd://bucketname
```

#### Upload Object

(1) first stage: create a new object on greenfield chain
```
gnfd-cmd  --config=config.toml  create-obj --contenType "text/xml"  gnfd://bucketname/objectname
```
(2) second stage: upload payload to greenfield storage provide

```
gnfd-cmd --config=config.toml  put --txnhash xxx  test.txt  gnfd://bucketname/objectname
```
required param:  --txnhash

#### Download Object

```
gnfd-cmd --config=config.toml  get gnfd://bucketname/objectname  test.txt  
```

#### Delete Bucket or Object
```
// delete bucekt:
gnfd-cmd --config=config.toml  del-bucket gnfd://bucketname

//delete object:
gnfd-cmd --config=config.toml  del-obj gnfd://bucketname/objectname
```
#### Head 

```
// head bucekt:
gnfd-cmd --config=config.toml  head-bucket gnfd://bucket-name

// head object:
gnfd-cmd --config=config.toml  head-obj gnfd://bucket-name/object-name
```

#### Compute Hash

```
gnfd-cmd get-hash --segSize 16  --dataShards 4 --parityShards 2 test.txt  
```

#### Challenge

```
gnfd-cmd  challenge --objectId "test" --pieceIndex 2  --spIndex -1
```
