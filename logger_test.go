package glog

import (
	"fmt"
	"runtime"
	"testing"
	"time"
	"sync"
)

func TestLogger(t *testing.T) {
	//t.Errorf("print some thing")
	// test Set value function.
	vt := 1000
	// log file abs path and relative path. can both set them.
	Set("path", "/tmp/glog", "mylog")
	// log file prefix name
	Set("name", "kawayi")
	Set("debug", false)
	// add format time string to each line of log. false: hidden, true:show
	// Set("show_time", false)
	// every new log file top content. ofcause it could be empty.
	Set("head", "auth@kawayi\nBbegin a new log every day\n")
	fmt.Println("DIR LOG -------------------:", log_mgr.log_file.fullpath(), log_mgr.log_file.dir_log)

	// test stack
	//var tmp_buf = stack(10000)
	//fmt.Println(string(tmp_buf))

	for i := 0; i < 4; i++ {
		_, file, line, ok := runtime.Caller(i)
		fmt.Println("RUN FILE", file, ":", line, ok)
	}

	fmt.Println("rotate_size:", log_mgr.log_file.rotate_size, "same_date:", log_mgr.log_file.checkRotate(time.Now()), "modify time:", getFileModifyTime(log_mgr.log_file.fullname(zero_time)) )
	wg := sync.WaitGroup{}

	wg.Add(2)
	go concurrencyLog(vt, &wg)
	go concurrencyLog(vt, &wg)
	wg.Wait()
	// 为了校验 删除过期日志
	time.Sleep(time.Second * 5)
}

func TestLetterCode(t *testing.T) {

	Set("path", "/tmp/glog", "mylog")
	Set("name", "kawayi")
	Set("head", "auth@kawayi\nBbegin a new log every day\n")
	Warnning("先弄点中文出来 log!")
	wg := sync.WaitGroup{}
	abssend := func() {
		for i := 0; i < 1000; i++ {
			Info("Oh my god. you are so clever")
			//Error("================an Error log!")
			Warnning("这是要追加在末尾的话 log!")
			Error("================an Error log!")
		}
		wg.Done()
	}
	wg.Add(1)
	go abssend()
	wg.Add(1)
	go abssend()
	wg.Wait()
}

// =================================function===============================

func concurrencyLog(send_time int, wg *sync.WaitGroup) {
	t1 := time.Now()
	for i := 0; i < send_time; i++ {
		Debug("testing something.")
		Info("Oh my god. you are so clever.")
		Warnning("这是要追加在末尾的话 log!")
		Error("================an Error log!")
		ErrorDepth(0, "this an user defined depth error log.")
	}
	elaps := time.Since(t1)
	fmt.Println("[PRINT][FINISH][SEND_TIME]", elaps)

	Debugf("testing something.")
	Infof("Oh my god. you are so clever.")
	Warnningf("这是要追加在末尾的话 log!")
	Errorf("================an Error log!")
	Fatalf("================an Error log!")

	Debugln("testing something.")
	Infoln("Oh my god. you are so clever.")
	Warnningln("这是要追加在末尾的话 log!")
	Errorln("================an Error log!")
	Fatalln("================an Error log!")
	wg.Done()
}
