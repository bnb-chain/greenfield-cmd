# greenfield-cmd

Greenfield client cmd tool, support api and cmd to make request to GreenField-StorageProvider and GreenField-BlockChain

## Cmd usage

```
// build:
cd cmd; go build -o gnfd-cmd main.go cmd_mb.go client_gnfd.go   cmd_upload.go  cmd_download.go 
 
// make bucket:
(1) gnfd-cmd pre-mb gnfd://bucketname
(2) send txn to chain use comsos client
    
// putObject:
 
(1) gnfd-cmd create-obj gnfd://bucketname/objectname
(2) send-cmd txn to chain use comsos client
(3) gnfd-cmd put --txnhash xxx  test.txt  gnfd://bucketname/objectname


// download:
gnfd-cmd  get gnfd://bucketname/objectname  test.txt  


// compute hash
./gnfd-cmd get-hash --segSize 16  --shards 6 test.txt  

```
