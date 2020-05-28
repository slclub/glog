package glog

import (
	"fmt"
	"testing"
)

func Benchmark_test(B *testing.B) {
	Set("path", "/tmp/glog", "mylog")
	// log file prefix name
	Set("name", "kawayi")
	// add format time string to each line of log. false: hidden, true:show
	// Set("show_time", false)
	// every new log file top content. ofcause it could be empty.
	Set("head", "auth@kawayi\nBbegin a new log every day\n")
	fmt.Println("DIR LOG -------------------:", log_mgr.log_file.fullpath(), log_mgr.log_file.dir_log)

	B.ReportAllocs()
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		Debug("testing something.")
		//Info("Oh my god. you are so clever.")
		//Warnning("这是要追加在末尾的话 log!")
		//Error("================an Error log!")
		//ErrorDepth(0, "this an user defined depth error log.")

	}

}
