package raft

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"
)

// Debugging
const Debug = 0

func DPrintln(a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Println(a...)
	}
	return
}

func DPrintf(format string, a ...interface{}) {
	if Debug > 0 {
		_, path, lineno, ok := runtime.Caller(1)
		_, file := filepath.Split(path)

		if ok {
			t := time.Now()
			a = append([]interface{}{t.Format("2006-01-02 15:04:05.00"), file, lineno}, a...)
			fmt.Printf("%s [%s:%d] "+format+"\n", a...)
		}
	}
}

func Min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func Max(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func Assert(flag bool, format string, a ...interface{}) {
	if !flag {
		_, path, lineno, ok := runtime.Caller(1)
		_, file := filepath.Split(path)

		if ok {
			t := time.Now()
			a = append([]interface{}{t.Format("2006-01-02 15:04:05.00"), file, lineno}, a...)
			reason := fmt.Sprintf("%s [%s:%d] "+format+"\n", a...)
			panic(reason)
		} else {
			panic("")
		}
	}
}

func FindMax(array []int) int {
	var max int = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
	}
	return max
}
