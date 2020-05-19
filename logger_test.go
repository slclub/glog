package glog

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	//t.Errorf("print some thing")
	// test Set value function.
	vt := 1000
	// log file abs path and relative path. can both set them.
	Set("path", "", "mylog")
	// log file prefix name
	Set("name", "kawayi")
	// add format time string to each line of log. false: hidden, true:show
	// Set("show_time", false)
	// every new log file top content. ofcause it could be empty.
	Set("head", "auth@kawayi\nBbegin a new log every day\n")
	fmt.Println("DIR LOG -------------------:", log_mgr.log_file.fullpath(), log_mgr.log_file.dir_log)

	// test stack
	var tmp_buf = stack(10000)
	fmt.Println(string(tmp_buf))

	for i := 0; i < 4; i++ {
		_, file, line, ok := runtime.Caller(3)
		fmt.Println("RUN FILE", file, ":", line, ok)
	}

	go concurrencyLog(vt)
	//go concurrencyLog(vt)
	time.Sleep(36000 * time.Second)
}

// =================================function===============================

func concurrencyLog(send_time int) {
	t1 := time.Now()
	for i := 0; i < send_time; i++ {
		Debug("testing something.")
		Info("Oh my god. you are so clever.")
		Warnning("an waring log!")
	}
	elaps := time.Since(t1)
	fmt.Println("[PRINT][FINISH][SEND_TIME]", elaps)
}
