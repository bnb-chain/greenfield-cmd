# greenfield-cmd

Greenfield client cmd tool, support api and cmd to make request to GreenField-StorageProvider and GreenField-BlockChain

## build

```
cd cmd; go build -o gnfd-cmd main.go cmd_mb.go client_gnfd.go  cmd_upload.go  cmd_download.go
cmd_hash.go  cmd_delete.go cmd_head.go

```
## Cmd usage

### basic config

```
cmd % cat config.toml 
endpoint = "http://127.0.0.1:8888"
host = "nodereal.gnfd.com"
grpcAddr = "http://127.0.0.1:26750"
chainId = "greenfield_9000-1741"% 

```

### support commands

```
   mb           create a new bucket
   put          upload object payload
   get          download object
   create-obj   create a new object
   get-hash     compute hash roots of object 
   del-obj      delete a existing object
   del-bucket   delete a existing bucket
   head-obj     headObject and get objectInfo
   head-bucket  headBucket and get bucketInfo
   help, h      Shows a list of commands or help for one command
```

### Create Bucket

create bucket: create a new bucket on greenfield chain
```
(1) gnfd-cmd --config=config.toml mb --primarySP "test-account" gnfd://bucketname
```

required param:  --primarySP

### Upload Object

(1) first stage: create a new object on greenfield chain
```
   gnfd-cmd  --config=config.toml  create-obj --contenType "text/xml"  gnfd://bucketname/objectname
```
(2) second stage: upload payload to greenfield storage provide

```
   gnfd-cmd --config=config.toml  put --txnhash xxx  test.txt  gnfd://bucketname/objectname
```
required param:  --txnhash

### Download Object

```
gnfd-cmd --config=config.toml  get gnfd://bucketname/objectname  test.txt  
```

### Delete Bucket or Object
```
// delete bucekt:
gnfd-cmd --config=config.toml  del-bucket gnfd://bucketname

//delete object:
gnfd-cmd --config=config.toml  del-obj gnfd://bucketname/objectname
```
### Head 

```
// head bucekt:
gnfd --config=config.toml  head-bucket gnfd://bucket-name

// head object:
gnfd --config=config.toml  head-obj gnfd://bucket-name/object-name
```

### Compute Hash

```
./gnfd-cmd get-hash --segSize 16  --dataShards 4 --parityShards 2 test.txt  
```
