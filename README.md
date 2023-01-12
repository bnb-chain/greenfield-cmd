# greenfield-sdk-go

## cmd usage

```
// build:
cd cmd; go build -o bfs main.go cmd_mb.go client_gnfd.go   cmd_upload.go  cmd_download.go 
 
// make bucket:
(1) gnfd pre-mb s3://bucketname
(2) send txn to chain use comsos client
(3) gnfd mb s3://bucketname  
    
// putObject:
 
(1) gnfd pre-upload s3://bucketname/objectname
(2) send txn to chain use comsos client
(3) gnfd put --txnhash xxx  test.txt  s3://bucketname/objectname


// download:
gnfd  get s3://bucketname/objectname  test.txt  

```
