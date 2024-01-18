package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	gtypes "github.com/bnb-chain/greenfield/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
)

// cmdPutObj return the command to finish uploading payload of the object
func cmdPutObj() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Action:    putObject,
		Usage:     "create object on chain and upload payload of object to SP",
		ArgsUsage: "[filePath]...  OBJECT-URL",
		Description: `
Send createObject txn to chain and upload the payload of object to the storage provider.
The command need to pass the file path inorder to compute hash roots on client.
Note that the  uploading with recursive flag only support folder.

Examples:
# create object and upload file to storage provider, the corresponding object is gnfd-object
$ gnfd-cmd object put file.txt gnfd://gnfd-bucket/gnfd-object,
# upload the files inside the folders
$ gnfd-cmd object put --tags='[{"key":"key1","value":"value1"},{"key":"key2","value":"value2"}]' --recursive folderName gnfd://bucket-name`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  secondarySPFlag,
				Value: "",
				Usage: "indicate the Secondary SP addr string list, input like addr1,addr2,addr3",
			},
			&cli.StringFlag{
				Name:  contentTypeFlag,
				Value: "",
				Usage: "indicate object content-type",
			},
			&cli.GenericFlag{
				Name: visibilityFlag,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: inheritType,
				},
				Usage: "set visibility of the object",
			},
			&cli.Uint64Flag{
				Name: partSizeFlag,
				// the default part size is 32M
				Value: 32 * 1024 * 1024,
				Usage: "indicate the resumable upload 's part size, uploading a large file in multiple parts. " +
					"The part size is an integer multiple of the segment size.",
			},
			&cli.BoolFlag{
				Name:  resumableFlag,
				Value: false,
				Usage: "indicate whether need to enable resumeable upload. Resumable upload refers to the process of uploading " +
					"a file in multiple parts, where each part is uploaded separately.This allows the upload to be resumed from " +
					"where it left off in case of interruptions or failures, rather than starting the entire upload process from the beginning.",
			},
			&cli.BoolFlag{
				Name:  recursiveFlag,
				Value: false,
				Usage: "performed on all files or objects under the specified directory or prefix in a recursive way",
			},
			&cli.BoolFlag{
				Name:  bypassSealFlag,
				Value: false,
				Usage: "if set this flag as true, it will not wait for the file to be sealed after the uploading is completed.",
			},
			&cli.StringFlag{
				Name:  tagFlag,
				Value: "",
				Usage: "set one or more tags of the object. The tag value is key-value pairs in json array format. E.g. [{\"key\":\"key1\",\"value\":\"value1\"},{\"key\":\"key2\",\"value\":\"value2\"}]",
			},
		},
	}
}

// cmdGetObj return the command to finish downloading object payload
func cmdGetObj() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Action:    getObject,
		Usage:     "download an object",
		ArgsUsage: "[filePath] OBJECT-URL",
		Description: `
Download a specific object from storage provider

Examples:
# download an object payload to file
$ gnfd-cmd object get gnfd://gnfd-bucket/gnfd-object  file.txt `,
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:  startOffsetFlag,
				Value: 0,
				Usage: "start offset info of the download body",
			},
			&cli.Int64Flag{
				Name:  endOffsetFlag,
				Value: 0,
				Usage: "end offset info of the download body",
			},
			&cli.Uint64Flag{
				Name: partSizeFlag,
				// the default part size is 32M
				Value: 32 * 1024 * 1024,
				Usage: "indicate the resumable upload 's part size, uploading a large file in multiple parts. " +
					"The part size is an integer multiple of the segment size.",
			},
			&cli.BoolFlag{
				Name:  resumableFlag,
				Value: false,
				Usage: "indicate whether need to enable resumeable download. Resumable download refers to the process of download " +
					"a file in multiple parts, where each part is downloaded separately.This allows the download to be resumed from " +
					"where it left off in case of interruptions or failures, rather than starting the entire download process from the beginning.",
			},
			&cli.StringFlag{
				Name:  spEndpointFlag,
				Value: "",
				Usage: "indicate object sp endpoint",
			},
		},
	}
}

// cmdCancelObjects cancel the object which has been created
func cmdCancelObjects() *cli.Command {
	return &cli.Command{
		Name:      "cancel",
		Action:    cancelCreateObject,
		Usage:     "cancel the created object",
		ArgsUsage: "OBJECT-URL",
		Description: `
Cancel the created object 

Examples:
$ gnfd-cmd object cancel  gnfd://gnfd-bucket/gnfd-object`,
	}
}

// cmdListObjects list the objects of the bucket
func cmdListObjects() *cli.Command {
	return &cli.Command{
		Name:      "ls",
		Action:    listObjects,
		Usage:     "list objects of the bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
List Objects of the bucket, including object name, object id, object status

Examples:
$ gnfd-cmd object ls gnfd://gnfd-bucket`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  recursiveFlag,
				Value: false,
				Usage: "performed on all files or objects under the specified directory or prefix in a recursive way",
			},
		},
	}
}

// cmdUpdateObject update the visibility of the object
func cmdUpdateObject() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Action:    updateObject,
		Usage:     "update object visibility",
		ArgsUsage: "OBJECT-URL",
		Description: `
Update the visibility of the object.
The visibility value can be public-read, private or inherit.

Examples:
update visibility of the gnfd-object
$ gnfd-cmd object update --visibility=public-read  gnfd://gnfd-bucket/gnfd-object`,
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name: visibilityFlag,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: privateType,
				},
				Usage: "set visibility of the bucket",
			},
		},
	}
}

// cmdGetUploadProgress return the uploading progress info of the object
func cmdGetUploadProgress() *cli.Command {
	return &cli.Command{
		Name:      "get-progress",
		Action:    getUploadInfo,
		Usage:     "get the uploading progress info of object",
		ArgsUsage: "OBJECT-URL",
		Description: `
The command is used to get the uploading progress info. 
you can use this command to view the progress information during the process of uploading a file to a Storage Provider.

Examples:
$ gnfd-cmd object get-progress gnfd://gnfd-bucket/gnfd-object`,
	}
}

func cmdMirrorObject() *cli.Command {
	return &cli.Command{
		Name:      "mirror",
		Action:    mirrorObject,
		Usage:     "mirror object to BSC",
		ArgsUsage: "",
		Description: `
Mirror a object as NFT to BSC

Examples:
# Mirror a object using object id
$ gnfd-cmd object mirror --destChainId 97 --id 1

# Mirror a object using bucket and object name
$ gnfd-cmd object mirror --destChainId 97 --bucketName yourBucketName --objectName yourObjectName
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     DestChainIdFlag,
				Value:    "",
				Usage:    "target chain id",
				Required: true,
			},
			&cli.StringFlag{
				Name:     IdFlag,
				Value:    "",
				Usage:    "object id",
				Required: false,
			},
			&cli.StringFlag{
				Name:     bucketNameFlag,
				Value:    "",
				Usage:    "bucket name",
				Required: false,
			},
			&cli.StringFlag{
				Name:     objectNameFlag,
				Value:    "",
				Usage:    "object name",
				Required: false,
			},
		},
	}
}

func cmdSetTagForObject() *cli.Command {
	return &cli.Command{
		Name:      "setTag",
		Action:    setTagForObject,
		Usage:     "Set tags for the given object",
		ArgsUsage: "OBJECT-URL",
		Description: `
The command is used to set tag for a given existing object.

Examples:
$ gnfd-cmd object setTag --tags='[{"key":"key1","value":"value1"},{"key":"key2","value":"value2"}]' gnfd://gnfd-bucket/gnfd-object`,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  tagFlag,
				Value: "",
				Usage: "set one or more tags for the given object. The tag value is key-value pairs in json array format. E.g. [{\"key\":\"key1\",\"value\":\"value1\"},{\"key\":\"key2\",\"value\":\"value2\"}]",
			},
		},
	}
}

// setTag Set tag for a given existing object
func setTagForObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	urlInfo := ctx.Args().First()
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}

	grn := gtypes.NewObjectGRN(bucketName, objectName)

	tagsParam := ctx.String(tagFlag)
	if tagsParam == "" {
		err = errors.New("invalid tags parameter")
	}
	if err != nil {
		return toCmdErr(err)
	}
	tags := &storageTypes.ResourceTags{}
	err = json.Unmarshal([]byte(tagsParam), &tags.Tags)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()
	txnHash, err := client.SetTag(c, grn.String(), *tags, sdktypes.SetTagsOptions{})

	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txnHash, "SetTags")
	if err != nil {
		return toCmdErr(err)
	}

	return nil
}

// putObject upload the payload of file, finish the third stage of putObject
func putObject(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return toCmdErr(fmt.Errorf("args number error"))
	}

	var (
		isUploadSingleFolder             bool
		bucketName, objectName, filePath string
		objectSize                       int64
		err                              error
		urlInfo                          string
	)

	gnfdClient, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return err
	}

	supportRecursive := ctx.Bool(recursiveFlag)

	if ctx.NArg() == 1 {
		// upload an empty folder
		urlInfo = ctx.Args().Get(0)
		bucketName, objectName, err = getObjAndBucketNames(urlInfo)
		if err != nil {
			return toCmdErr(err)
		}
		if strings.HasSuffix(objectName, "/") {
			isUploadSingleFolder = true
		} else {
			return toCmdErr(errors.New("no file path to upload, if you need create a folder, the folder name should be end with /"))
		}

		if err = uploadFile(bucketName, objectName, filePath, urlInfo, ctx, gnfdClient, isUploadSingleFolder, true, 0); err != nil {
			return toCmdErr(err)
		}

	} else {
		// upload files in folder in a recursive way
		if supportRecursive {
			urlInfo = ctx.Args().Get(1)
			if err = uploadFolder(urlInfo, ctx, gnfdClient); err != nil {
				return toCmdErr(err)
			}
			return nil
		}

		filePathList := make([]string, 0)
		argNum := ctx.Args().Len()
		for i := 0; i < argNum-1; i++ {
			filePathList = append(filePathList, ctx.Args().Get(i))
		}

		var needUploadMutiFiles bool
		if len(filePathList) > 1 {
			needUploadMutiFiles = true
		}

		// upload multiple files
		if needUploadMutiFiles {
			urlInfo = ctx.Args().Get(argNum - 1)
			bucketName = ParseBucket(urlInfo)
			if bucketName == "" {
				return toCmdErr(errors.New("fail to parse bucket name"))
			}

			for idx, fileName := range filePathList {
				nameList := strings.Split(fileName, "/")
				objectName = nameList[len(nameList)-1]
				objectSize, err = parseFileByArg(ctx, idx)
				if err != nil {
					return toCmdErr(err)
				}

				if err = uploadFile(bucketName, objectName, fileName, urlInfo, ctx, gnfdClient, false, true, objectSize); err != nil {
					fmt.Println("upload object:", objectName, "err", err)
				}
				fmt.Println()
			}
		} else {
			// upload single file
			objectSize, err = parseFileByArg(ctx, 0)
			if err != nil {
				return toCmdErr(err)
			}
			urlInfo = ctx.Args().Get(1)
			bucketName, objectName, err = getObjAndBucketNames(urlInfo)
			if err != nil {
				bucketName = ParseBucket(urlInfo)
				if bucketName == "" {
					return toCmdErr(errors.New("fail to parse bucket name"))
				}
				// if the object name has not been set, set the file name as object name
				objectName = filepath.Base(filePathList[0])
			}
			if err = uploadFile(bucketName, objectName, filePathList[0], urlInfo, ctx, gnfdClient, false, true, objectSize); err != nil {
				return toCmdErr(err)
			}
		}
	}

	return nil
}

// uploadFolder upload folder and the files inside to bucket in a recursive way
func uploadFolder(urlInfo string, ctx *cli.Context,
	gnfdClient client.IClient) error {
	// upload folder with recursive flag
	bucketName := ParseBucket(urlInfo)
	if bucketName == "" {
		return errors.New("fail to parse bucket name")
	}

	folderName := ctx.Args().Get(0)
	fileInfo, err := os.Stat(folderName)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return errors.New("failed to parse folder path with recursive flag")
	}

	homeDir, err := getHomeDir(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	taskID := uuid.New().String()

	fmt.Println("================================================")
	fmt.Println("Your batch upload is submitted as a task, task ID is " + taskID)
	fmt.Println("You can check your task status and progress by using cmd as below:\n\n- List all your tasks: ./gnfd-cmd task ls\n- Check status: ./gnfd-cmd task status --task.id taskID\n- Retry (in case this process is killed accidentally): ./gnfd-cmd task retry --task.id taskID\n- Delete task: ...\n\n>>================================================")
	fmt.Println("Upload Task building")

	taskState := &TaskState{
		Lock:        new(sync.Mutex),
		ObjectState: make(map[int]*UploadTaskObject),
		TaskID:      taskID,
		FolderName:  folderName,
		BucketName:  bucketName,
		Status:      TaskStatusCreate,
	}

	baseDir := filepath.Base(folderName)

	fileInfos := make([]os.FileInfo, 0)
	filePaths := make([]string, 0)
	objectNames := make([]string, 0)
	objectIndex := 0

	listFolderErr := filepath.Walk(folderName, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fileInfos = append(fileInfos, info)
			// use the base dir to construct object name of the file
			index := strings.Index(path, baseDir)
			if index == notFound {
				return nil
			}
			objectNames = append(objectNames, path[index:])
			filePaths = append(filePaths, path)
		} else {
			// use the base dir to construct object name of the sub-folder
			index := strings.Index(path, baseDir)
			if index == notFound {
				return nil
			}
			subFolderName := path[index:] + "/"
			utj := &UploadTaskObject{
				BucketName:         bucketName,
				ObjectName:         subFolderName,
				FilePath:           path,
				UploadSingleFolder: true,
				ObjectSize:         0,
				Status:             TaskObjectStatusWaitForUpload,
			}
			taskState.ObjectState[objectIndex] = utj
			objectIndex++
		}
		return nil
	})

	if listFolderErr != nil {
		return listFolderErr
	}
	// upload folder
	for id, info := range fileInfos {
		utj := &UploadTaskObject{
			BucketName:         bucketName,
			ObjectName:         objectNames[id],
			FilePath:           filePaths[id],
			UploadSingleFolder: false,
			ObjectSize:         info.Size(),
			Status:             TaskObjectStatusWaitForUpload,
		}
		taskState.ObjectState[objectIndex] = utj
		objectIndex++
	}

	// store tag
	contentType := ctx.String(contentTypeFlag)
	secondarySPAccs := ctx.String(secondarySPFlag)
	partSize := ctx.Uint64(partSizeFlag)
	tags := ctx.String(tagFlag)
	visibility := ctx.Generic(visibilityFlag)

	taskState.Flag = UploadFlag{
		ContentType: contentType,
		SecondarySP: secondarySPAccs,
		PartSize:    partSize,
		Tags:        tags,
		Visibility:  storageTypes.VISIBILITY_TYPE_INHERIT,
	}

	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		taskState.Flag.Visibility = visibilityTypeVal
	}

	taskFileName := fmt.Sprintf("/.%s/state", taskID)
	taskFilePath := filepath.Join(homeDir, taskFileName)

	content, err := json.Marshal(taskState)
	if err != nil {
		return toCmdErr(err)
	}

	err = createAndWriteFile(taskFilePath, content)
	if err != nil {
		return err
	}

	return uploadFolderByTask(ctx, homeDir, gnfdClient, taskState)
}

func uploadFolderByTask(ctx *cli.Context, homeDir string, gnfdClient client.IClient, taskState *TaskState) error {
	signalCtx, cancel := context.WithCancel(ctx.Context)
	go stateSync(signalCtx, homeDir, taskState)
	sealSignal := make(chan int)
	go sealChecker(ctx, taskState, gnfdClient, sealSignal)

	for index, object := range taskState.ObjectState {
		// check object status
		if object.Status == TaskObjectStatusCreated || object.Status == TaskObjectStatusSeal {
			continue
		}

		err := uploadFileByTask(object.BucketName, object.ObjectName, object.FilePath, taskState.Flag, gnfdClient, object.UploadSingleFolder, object.ObjectSize)
		if err != nil {
			taskState.UpdateObjectState(index, TaskObjectStatusFailed, err.Error())
			fmt.Printf("\r%s", fmt.Sprintf("%s %s %s", TaskObjectStatusFailed, object.ObjectName, err.Error()))
		} else {
			taskState.UpdateObjectState(index, TaskObjectStatusCreated, "")
			fmt.Printf("\r%s", fmt.Sprintf("%s %s %s", TaskObjectStatusCreated, object.ObjectName, ""))
		}
	}

	// waiting for seal
	<-sealSignal
	cancel()
	// waiting sync status
	time.Sleep(5 * time.Second)
	fmt.Println()
	if taskState.Status != TaskStatusSuccess {
		taskState.Status = TaskStatusFail
	}
	content, err := json.Marshal(taskState)
	if err != nil {
		return toCmdErr(err)
	}
	taskFileName := fmt.Sprintf("/.%s/state", taskState.TaskID)
	taskFilePath := filepath.Join(homeDir, taskFileName)
	if err = createAndWriteFile(taskFilePath, content); err != nil {
		return toCmdErr(err)
	}
	return nil
}

// sealChecker Wait for seal and update the status
func sealChecker(ctx *cli.Context, taskState *TaskState, gnfdClient client.IClient, signal chan int) {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	objectOffset := 0
	for range tick.C {
		if objectOffset >= len(taskState.ObjectState) {
			break
		}
		utj := taskState.ObjectState[objectOffset]
		headObjOutput, queryErr := gnfdClient.HeadObject(ctx.Context, utj.BucketName, utj.ObjectName)
		if queryErr != nil {
			continue
		}

		if headObjOutput.ObjectInfo.GetObjectStatus().String() == "OBJECT_STATUS_SEALED" {
			taskState.UpdateObjectState(objectOffset, TaskObjectStatusSeal, "")
			fmt.Printf("\r%s", fmt.Sprintf("%s %s %s", TaskObjectStatusSeal, utj.ObjectName, ""))
			objectOffset++
		}
	}

	taskState.Status = TaskStatusSuccess
	signal <- 1
}

// stateSync Periodically synchronize the status to the local file
func stateSync(ctx context.Context, homeDir string, state *TaskState) {
	taskFileName := fmt.Sprintf("/.%s/state", state.TaskID)
	taskFilePath := filepath.Join(homeDir, taskFileName)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			content, err := json.Marshal(state)
			if err != nil {
				continue
			}
			_ = createAndWriteFile(taskFilePath, content)
		case <-ctx.Done():
			return
		}
	}
}

func uploadFile(bucketName, objectName, filePath, urlInfo string, ctx *cli.Context,
	gnfdClient client.IClient, uploadSingleFolder, printTxnHash bool, objectSize int64) error {
	var file *os.File
	contentType := ctx.String(contentTypeFlag)
	secondarySPAccs := ctx.String(secondarySPFlag)
	partSize := ctx.Uint64(partSizeFlag)
	resumableUpload := ctx.Bool(resumableFlag)
	bypassSeal := ctx.Bool(bypassSealFlag)

	opts := sdktypes.CreateObjectOptions{}

	tags := ctx.String(tagFlag)
	if tags != "" {
		opts.Tags = &storageTypes.ResourceTags{}
		err := json.Unmarshal([]byte(tags), &opts.Tags.Tags)
		if err != nil {
			return toCmdErr(err)
		}
	}

	if contentType != "" {
		opts.ContentType = contentType
	} else {
		// parse the mimeType as content type
		mimeType, err := getContentTypeOfFile(filePath)
		if err == nil {
			opts.ContentType = mimeType
		}
	}

	visibity := ctx.Generic(visibilityFlag)
	if visibity != "" {
		visibityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibity))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibityTypeVal
	}

	// set second sp address if provided by user
	if secondarySPAccs != "" {
		secondarySplist := strings.Split(secondarySPAccs, ",")
		addrList := make([]sdk.AccAddress, len(secondarySplist))
		for idx, addr := range secondarySplist {
			addrList[idx] = sdk.MustAccAddressFromHex(addr)
		}
		opts.SecondarySPAccs = addrList
	}

	c, cancelPutObject := context.WithCancel(globalContext)
	defer cancelPutObject()

	_, err := gnfdClient.HeadObject(c, bucketName, objectName)
	var txnHash string
	// if err==nil, object exist on chain, no need to createObject
	if err != nil {
		if uploadSingleFolder {
			txnHash, err = gnfdClient.CreateFolder(c, bucketName, objectName, opts)
			if err != nil {
				return toCmdErr(err)
			}
		} else {
			// Open the referenced file.
			file, err = os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			txnHash, err = gnfdClient.CreateObject(c, bucketName, objectName, file, opts)
			if err != nil {
				return toCmdErr(err)
			}
		}
		if printTxnHash {
			fmt.Printf("object %s created on chain \n", objectName)
			fmt.Println("transaction hash: ", txnHash)
		}
	} else {
		fmt.Printf("object %s already exist \n", objectName)
	}

	if objectSize == 0 {
		return nil
	}

	opt := sdktypes.PutObjectOptions{}
	if contentType != "" {
		opt.ContentType = contentType
	}

	opt.DisableResumable = !resumableUpload
	opt.PartSize = partSize

	// Open the referenced file.
	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// if the file is more than 2G , it needs to force use resume uploading
	if objectSize > maxPutWithoutResumeSize {
		opt.DisableResumable = false
	}

	progressReader := &ProgressReader{
		Reader:      reader,
		Total:       objectSize,
		StartTime:   time.Now(),
		LastPrinted: time.Now(),
	}

	// if print big file progress, the printing progress should be delayed to obtain a more accurate display.
	if objectSize > progressDelayPrintSize {
		progressReader.LastPrinted = time.Now().Add(3 * time.Second)
	}

	if opt.DisableResumable {
		if err = gnfdClient.PutObject(c, bucketName, objectName,
			objectSize, progressReader, opt); err != nil {
			return toCmdErr(err)
		}
	} else {
		fmt.Printf("resumable uploading %s is beginning...\n", objectName)
		if err = gnfdClient.PutObject(c, bucketName, objectName,
			objectSize, progressReader, opt); err != nil {
			return toCmdErr(err)
		}
	}

	if bypassSeal {
		fmt.Printf("\nupload %s to %s \n", objectName, urlInfo)
		return nil
	}

	// Check if object is sealed
	timeout := time.After(1 * time.Hour)
	ticker := time.NewTicker(3 * time.Second)
	count := 0
	fmt.Println()
	fmt.Println("sealing...")
	for {
		select {
		case <-timeout:
			return toCmdErr(errors.New("object not sealed after one hour"))
		case <-ticker.C:
			count++
			headObjOutput, queryErr := gnfdClient.HeadObject(c, bucketName, objectName)
			if queryErr != nil {
				return queryErr
			}
			if count%10 == 0 {
				fmt.Println("sealing...")
			}
			if headObjOutput.ObjectInfo.GetObjectStatus().String() == "OBJECT_STATUS_SEALED" {
				ticker.Stop()
				fmt.Printf("upload %s to %s \n", objectName, urlInfo)
				return nil
			}
		}
	}
}

func uploadFileByTask(bucketName, objectName, filePath string, uploadFlag UploadFlag,
	gnfdClient client.IClient, uploadSingleFolder bool, objectSize int64) error {
	var file *os.File

	opts := sdktypes.CreateObjectOptions{}

	if uploadFlag.Tags != "" {
		opts.Tags = &storageTypes.ResourceTags{}
		err := json.Unmarshal([]byte(uploadFlag.Tags), &opts.Tags.Tags)
		if err != nil {
			return toCmdErr(err)
		}
	}

	if uploadFlag.ContentType != "" {
		opts.ContentType = uploadFlag.ContentType
	} else {
		// parse the mimeType as content type
		mimeType, err := getContentTypeOfFile(filePath)
		if err == nil {
			opts.ContentType = mimeType
		}
	}

	opts.Visibility = uploadFlag.Visibility

	// set second sp address if provided by user
	if uploadFlag.SecondarySP != "" {
		secondarySplist := strings.Split(uploadFlag.SecondarySP, ",")
		addrList := make([]sdk.AccAddress, len(secondarySplist))
		for idx, addr := range secondarySplist {
			addrList[idx] = sdk.MustAccAddressFromHex(addr)
		}
		opts.SecondarySPAccs = addrList
	}

	c, cancelPutObject := context.WithCancel(globalContext)
	defer cancelPutObject()

	_, err := gnfdClient.HeadObject(c, bucketName, objectName)

	// if err==nil, object exist on chain, no need to createObject
	if err != nil {
		if uploadSingleFolder {
			_, err = gnfdClient.CreateFolder(c, bucketName, objectName, opts)
			if err != nil {
				return toCmdErr(err)
			}
		} else {
			// Open the referenced file.
			file, err = os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = gnfdClient.CreateObject(c, bucketName, objectName, file, opts)
			if err != nil {
				return toCmdErr(err)
			}
		}
	}

	if objectSize == 0 {
		return nil
	}

	opt := sdktypes.PutObjectOptions{}
	if uploadFlag.ContentType != "" {
		opt.ContentType = uploadFlag.ContentType
	}

	opt.PartSize = uploadFlag.PartSize

	// Open the referenced file.
	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// if the file is more than 2G , it needs to force use resume uploading
	if objectSize > maxPutWithoutResumeSize {
		opt.DisableResumable = false
	}

	progressReader := &ProgressReader{
		Reader:      reader,
		Total:       objectSize,
		StartTime:   time.Now(),
		LastPrinted: time.Now(),
	}

	// if print big file progress, the printing progress should be delayed to obtain a more accurate display.
	if objectSize > progressDelayPrintSize {
		progressReader.LastPrinted = time.Now().Add(3 * time.Second)
	}
	if err = gnfdClient.PutObject(c, bucketName, objectName,
		objectSize, progressReader, opt); err != nil {
		return toCmdErr(err)
	}

	return nil
}

// getObject download the object payload from sp
func getObject(ctx *cli.Context) error {
	var err error
	if ctx.NArg() < 1 {
		return toCmdErr(fmt.Errorf("args number less than one"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	spEndpoint := ctx.String(spEndpointFlag)

	gnfdClient, err := NewClient(ctx, ClientOptions{IsQueryCmd: false, ForceToUseSpecifiedSpEndpointForDownloadOnly: spEndpoint})
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelGetObject := context.WithCancel(globalContext)
	defer cancelGetObject()

	var filePath string
	if ctx.Args().Len() == 1 {
		filePath = objectName
	} else if ctx.Args().Len() == 2 {
		filePath = ctx.Args().Get(1)
		stat, err := os.Stat(filePath)
		if err == nil {
			if stat.IsDir() {
				if strings.HasSuffix(filePath, "/") {
					filePath += objectName
				} else {
					filePath = filePath + "/" + objectName
				}
			}
		}
	}

	filePath, err = checkIfDownloadFileExist(filePath, objectName)
	if err != nil {
		return toCmdErr(err)
	}

	opt := sdktypes.GetObjectOptions{}
	startOffset := ctx.Int64(startOffsetFlag)
	endOffset := ctx.Int64(endOffsetFlag)
	partSize := ctx.Uint64(partSizeFlag)
	resumableDownload := ctx.Bool(resumableFlag)

	// flag has been set
	if startOffset != 0 || endOffset != 0 {
		if err = opt.SetRange(startOffset, endOffset); err != nil {
			return toCmdErr(err)
		}
	}

	if resumableDownload {
		opt.PartSize = partSize
		err = gnfdClient.FGetObjectResumable(c, bucketName, objectName, filePath, opt)
		if err != nil {
			return toCmdErr(err)
		}
		fmt.Printf("resumable download object %s, the file path is %s \n", objectName, filePath)
	} else {
		var fd *os.File
		dir := filepath.Dir(filePath)
		fileName := "." + filepath.Base(filePath) + ".tmp"
		tempFilePath := filepath.Join(dir, fileName)

		tempFilePath, err = checkIfDownloadFileExist(tempFilePath, objectName)
		if err != nil {
			return toCmdErr(err)
		}
		// download to the temp file firstly
		fd, err = os.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			return err
		}

		defer fd.Close()

		body, info, downloadErr := gnfdClient.GetObject(c, bucketName, objectName, opt)
		if downloadErr != nil {
			return toCmdErr(downloadErr)
		}

		pw := &ProgressWriter{
			Writer:      fd,
			Total:       info.Size,
			StartTime:   time.Now(),
			LastPrinted: time.Now(),
		}

		_, err = io.Copy(pw, body)
		if err != nil {
			return toCmdErr(err)
		}

		err = os.Rename(tempFilePath, filePath)
		if err != nil {
			fmt.Printf("failed to rename %s to %s \n", tempFilePath, filePath)
			return nil
		}
		fmt.Printf("\ndownload object %s, the file path is %s, content length:%d \n", objectName, filePath, uint64(info.Size))
	}

	return nil
}

// cancelCreateObject cancel the created object on chain
func cancelCreateObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	cli, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCancelCreate := context.WithCancel(globalContext)
	defer cancelCancelCreate()

	_, err = cli.HeadObject(c, bucketName, objectName)
	if err != nil {
		return toCmdErr(ErrObjectNotCreated)
	}

	_, err = cli.CancelCreateObject(c, bucketName, objectName, sdktypes.CancelCreateOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Println("cancel create object:", objectName)
	return nil
}

func listObjects(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	bucketName, prefixName, err := ParseBucketAndPrefix(ctx.Args().Get(0))
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}
	c, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	supportRecursive := ctx.Bool(recursiveFlag)
	err = listObjectByPage(client, c, bucketName, prefixName, supportRecursive)
	if err != nil {
		return toCmdErr(err)
	}

	return nil
}

func listObjectByPage(cli client.IClient, c context.Context, bucketName, prefixName string, isRecursive bool) error {
	var (
		listResult        sdktypes.ListObjectsResult
		continuationToken string
		err               error
	)

	for {
		if isRecursive {
			listResult, err = cli.ListObjects(c, bucketName, sdktypes.ListObjectsOptions{ShowRemovedObject: false,
				MaxKeys:           defaultMaxKey,
				ContinuationToken: continuationToken,
				Prefix:            prefixName})
		} else {
			listResult, err = cli.ListObjects(c, bucketName, sdktypes.ListObjectsOptions{ShowRemovedObject: false,
				Delimiter:         "/",
				MaxKeys:           defaultMaxKey,
				ContinuationToken: continuationToken,
				Prefix:            prefixName})
		}
		if err != nil {
			return toCmdErr(err)
		}

		printListResult(listResult)
		if !listResult.IsTruncated {
			break
		}

		continuationToken = listResult.NextContinuationToken
	}
	return nil
}

func printListResult(listResult sdktypes.ListObjectsResult) {
	for _, object := range listResult.Objects {
		info := object.ObjectInfo
		location, _ := time.LoadLocation("Asia/Shanghai")
		t := time.Unix(info.CreateAt, 0).In(location)

		fmt.Printf("%s %15d %s \n", t.Format(iso8601DateFormat), info.PayloadSize, info.ObjectName)
	}
	// list the folders
	for _, prefix := range listResult.CommonPrefixes {
		fmt.Printf("%s %15s %s \n", strings.Repeat(" ", len(iso8601DateFormat)), "PRE", prefix)
	}

}

func updateObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be one"))
	}

	urlInfo := ctx.Args().First()
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelUpdateObject := context.WithCancel(globalContext)
	defer cancelUpdateObject()

	visibility := ctx.Generic(visibilityFlag)
	if visibility == "" {
		return toCmdErr(fmt.Errorf("visibity must set to be updated"))
	}

	visibilityType, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
	if typeErr != nil {
		return typeErr
	}

	txnHash, err := client.UpdateObjectVisibility(c, bucketName, objectName, visibilityType, sdktypes.UpdateObjectOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		fmt.Println("update object visibility error:", err.Error())
		return nil
	}

	err = waitTxnStatus(client, c, txnHash, "UpdateObject")
	if err != nil {
		return toCmdErr(err)
	}

	objectDetail, err := client.HeadObject(c, bucketName, objectName)
	if err != nil {
		// head fail, no need to print the error
		return nil
	}

	fmt.Printf("update object visibility finished, latest object visibility:%s\n", objectDetail.ObjectInfo.GetVisibility().String())
	fmt.Println("transaction hash: ", txnHash)
	return nil
}

func getUploadInfo(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number should be 1"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelGetUploadInfo := context.WithCancel(globalContext)
	defer cancelGetUploadInfo()

	uploadInfo, err := client.GetObjectUploadProgress(c, bucketName, objectName)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Println("uploading progress:", uploadInfo)
	return nil
}

func pathExists(path string) (bool, int64, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, 0, nil
	}

	if err == nil {
		if stat.IsDir() {
			return false, 0, fmt.Errorf("not support upload dir without recursive flag")
		}
		return true, stat.Size(), nil
	}

	return false, 0, err
}

func getObjAndBucketNames(urlInfo string) (string, string, error) {
	bucketName, objectName, err := ParseBucketAndObject(urlInfo)
	if bucketName == "" || objectName == "" || err != nil {
		return "", "", fmt.Errorf("fail to parse bucket name or object name")
	}
	return bucketName, objectName, nil
}

func mirrorObject(ctx *cli.Context) error {
	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}
	id := math.NewUint(0)
	if ctx.String(IdFlag) != "" {
		id = math.NewUintFromString(ctx.String(IdFlag))
	}
	destChainId := ctx.Int64(DestChainIdFlag)
	bucketName := ctx.String(bucketNameFlag)
	objectName := ctx.String(objectNameFlag)
	c, cancelContext := context.WithCancel(globalContext)
	defer cancelContext()

	txResp, err := client.MirrorObject(c, sdk.ChainID(destChainId), id, bucketName, objectName, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("mirror object succ, txHash: %s\n", txResp.TxHash)
	return nil
}
