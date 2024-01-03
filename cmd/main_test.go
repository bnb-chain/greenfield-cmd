package main

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func Test_Main(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"./gnfd-cmd", "--passwordfile", "../build/p.txt", "object", "get", "--fast", "gnfd://bnb-sp3/yellow-paper.pdf"}
	os.Args = []string{"./gnfd-cmd", "--passwordfile", "../build/p.txt", "object", "get", "gnfd://bnb-sp3/yellow-paper.pdf"}

	mainWithArg(os.Args)
	os.Args = oldArgs

	time.Sleep(5 * time.Second)
	//
	//var wg sync.WaitGroup
	//wg.Add(10)
	//for i := 0; i < 10; i++ {
	//	go work(&wg)
	//}
	//
	//wg.Wait()
	//// Wait to see the global run queue deplete.
	//time.Sleep(3 * time.Second)
}

func work(wg *sync.WaitGroup) {
	time.Sleep(time.Second)
	mem := make([]int, 1000)
	log.Println(mem)
	var counter int
	for i := 0; i < 1e5; i++ {
		time.Sleep(time.Millisecond * 100)
		counter++
	}
	wg.Done()
}
