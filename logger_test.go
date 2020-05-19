package glog

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	//t.Errorf("print some thing")
	vt := 1000
	_, file, line, ok := runtime.Caller(3 + 1)
	fmt.Println("RUN FILE", file, ":", line, ok)

	go concurrencyLog(vt)
	go concurrencyLog(vt)
	time.Sleep(36000 * time.Second)
}

// =================================function===============================

func concurrencyLog(send_time int) {
	for i := 0; i < send_time; i++ {
		Debug("testing something.")
		Info("Oh my god. you are so clever.")
		Warnning("an waring log!")
	}
	fmt.Println("[PRINT][FINISH]")
}
