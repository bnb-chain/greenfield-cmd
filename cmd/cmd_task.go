package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func cmdTaskStatus() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Action:    getTaskStatus,
		Usage:     "get task status by task id",
		ArgsUsage: "",
		Description: `
Examples:
$ gnfd-cmd task status --taskId 123`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     taskIDFlag,
				Value:    "",
				Usage:    "task id",
				Required: true,
			},
		},
	}
}

func cmdTaskDelete() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Action:    deleteTask,
		Usage:     "delete task by id",
		ArgsUsage: "",
		Description: `
Examples:
$ gnfd-cmd task delete --taskId 123 `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     taskIDFlag,
				Value:    "",
				Usage:    "task id",
				Required: true,
			},
		},
	}
}

func cmdTaskRetry() *cli.Command {
	return &cli.Command{
		Name:      "retry",
		Action:    retryTask,
		Usage:     "retry task by id",
		ArgsUsage: "",
		Description: `
Examples:
$ gnfd-cmd task retry --taskId 123 `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     taskIDFlag,
				Value:    "",
				Usage:    "task id",
				Required: true,
			},
		},
	}
}

func getTaskStatus(ctx *cli.Context) error {
	content, err := getTaskState(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Folder: %s\n", content.FolderName)
	fmt.Printf("Status: %s\n", content.Status)
	for _, state := range content.ObjectState {
		fmt.Printf("%s\n", fmt.Sprintf("%s %s %s", state.Status, state.ObjectName, state.Comment))
	}
	fmt.Println()
	return nil
}

func deleteTask(ctx *cli.Context) error {
	taskID := ctx.String(taskIDFlag)
	homeDir, err := getHomeDir(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	taskFileName := fmt.Sprintf("/.%s", taskID)
	taskFilePath := filepath.Join(homeDir, taskFileName)
	if !fileExists(taskFilePath) {
		return toCmdErr(fmt.Errorf("task not found"))
	}
	err = os.RemoveAll(taskFilePath)
	if err != nil {
		return toCmdErr(err)
	}
	return nil
}

func retryTask(ctx *cli.Context) error {
	content, err := getTaskState(ctx)
	if err != nil {
		return err
	}
	homeDir, err := getHomeDir(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	gnfdClient, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return err
	}
	fmt.Printf("task: %s\n", content.TaskID)
	fmt.Printf("folder name: %s\n", content.FolderName)
	fmt.Println("retrying...")
	return uploadFolderByTask(ctx, homeDir, gnfdClient, content)
}

func getTaskState(ctx *cli.Context) (*TaskState, error) {
	taskID := ctx.String(taskIDFlag)
	homeDir, err := getHomeDir(ctx)
	if err != nil {
		return nil, toCmdErr(err)
	}
	taskFileName := fmt.Sprintf("/.%s/state", taskID)
	taskFilePath := filepath.Join(homeDir, taskFileName)
	if !fileExists(taskFilePath) {
		return nil, toCmdErr(fmt.Errorf("task not found"))
	}
	value, err := readFile(taskFilePath)
	if err != nil {
		return nil, toCmdErr(err)
	}
	var content TaskState
	err = json.Unmarshal(value, &content)
	if err != nil {
		return nil, toCmdErr(err)
	}
	return &content, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	return !os.IsNotExist(err)
}
