# greenfield-cmd

Greenfield client cmd tool, support commands to make request to greenfield

## build

```
cd cmd; go build -o gnfd-cmd main.go cmd_mb.go client_gnfd.go  cmd_upload.go  cmd_download.go
cmd_hash.go  cmd_delete.go cmd_head.go cmd_challenge.go

```
## Cmd usage

### basic config 

config file example
```
cmd % cat config.toml 
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
```
### Precautions

1.If the private key has not been configured, the tool will generate one and the operator address

2.The operator account should have balance before testing

### Examples
#### Create Bucket

create bucket: create a new bucket on greenfield chain
```
(1) gnfd-cmd --config=config.toml mb  gnfd://bucketname
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
gnfd --config=config.toml  head-bucket gnfd://bucket-name

// head object:
gnfd --config=config.toml  head-obj gnfd://bucket-name/object-name
```

#### Compute Hash

```
./gnfd-cmd get-hash --segSize 16  --dataShards 4 --parityShards 2 test.txt  
```

#### Challege

```
./gnfd-cmd  challenge --objectId "test" --pieceIndex 2  --spIndex -1
```