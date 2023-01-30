# greenfield-sdk-go

Greenfield Go SDK, support api and cmd to make request to GreenField-StorageProvider and GreenField-BlockChain

## Install 

go get -u github.com/bnb-chain/greenfield-sdk-go

## Cmd usage

```
// build:
cd cmd; go build -o gnfd main.go cmd_mb.go client_gnfd.go   cmd_upload.go  cmd_download.go 
 
// make bucket:
(1) gnfd pre-mb gnfd://bucketname
(2) send txn to chain use comsos client
(3) gnfd mb gnfd://bucketname  
    
// putObject:
 
(1) gnfd pre-upload gnfd://bucketname/objectname
(2) send txn to chain use comsos client
(3) gnfd put --txnhash xxx  test.txt  gnfd://bucketname/objectname


// download:
gnfd  get gnfd://bucketname/objectname  test.txt  

```
