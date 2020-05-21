# glog
golang logger manager

### Summary

Log plug-ins that can be customized appropriately. Support concurrent logging.We used the ring queue and sync.Pool .
and use queue instead of lock(sync.Mutex) write security. Easy to use.  Customized. Log files can be customized by size and date. Both supported.

### Install

`go get github.com/slclub/glog`

`go mod`


### Let's starts.

If you don't want set your path, debug, time and so on. It can still be used

- Import

`import "github.com/slclub/glog"`

- API

Like fmt.Pringln

```go
  glog.Info("[HELLO][WORLD]", "MY FIST START", "PID[", int , "]")
  glog.Debug("[HELLO][WORLD]", "MY FIST START", "PID[", int , "]")
  glog.Warnning("[HELLO][WORLD]", "MY FIST START", "PID[", int , "]")
  glog.Error("[HELLO][WORLD]", "MY FIST START", "PID[", int , "]")
  glog.Fatal("[HELLO][WORLD]", "MY FIST START", "PID[", int , "]")

```

- An example.

```go
func concurrencyLog(send_time int) {
    for i := 0; i < send_time; i++ {
        glog.Debug("testing something.")
        glog.Info("Oh my god. you are so clever.")
        glog.Warnning("an waring log!")
    }   
    fmt.Println("[PRINT][FINISH]")
}

func main() {
    glog.Set("path", "", "mylog")
    go concurrencyLog(1000)
    go concurrencyLog(1000)
    time.Sleep(1800 * time.Second)

}


```

- Log default style. before you customized

```go
 2020-05-20 00:20:38 INFO  Oh my god. you are so clever.
 2020-05-20 00:20:38 WARN  an waring log!
 2020-05-20 00:20:38 DEBUG testing something.
 2020-05-20 00:20:38 INFO  Oh my god. you are so clever.
 2020-05-20 00:20:38 WARN  an waring log!
 2020-05-20 00:20:38 DEBUG testing something.
 2020-05-20 00:20:38 INFO  Oh my god. you are so clever.
```

### Customized

It's actually a function

`Set(field string, value ...interface{})`

In you code maybe like `glog.Set(xxx, xxx)`

- Log file path setting.

```go
// log file abs path and relative path. can both set them.
// In fact, the two paths correspond to the project path and its path respectively
// You can choose only one of them to set.
Set("path", "abs_path", "rel_path")
```

- Log name prefix

```go
// We offen  concatenated log name  with string and time and random number.
// here you just set the string part of the name. 
Set("name", "kawayi")
```

- Log time settings

```go
// Need to hide the time of each log line 
// false : hidden, true: show ;  
// default format : 2020-50-19 00:00:00
Set("show_time", false)
```

- Log head info

```go
// Log files are changed. when a new file was created. this string you added will be set to top of file.
Set("head", "auth@kawayi\nBbegin a new log every day\n")
```

- Log debug settings.

```go
// false: Hide data printed by a Debug method. true: show all
Set("debug", false)
// The second param also can be a number. The permission setting here is designed by bit calculation
// Add and subtract are supported for these numbers. You can use addition and subtraction to control permissions
// Very delicate to control
Set("debug", int)

const (
    // LEVEL
    LEVEL_INFO     = 1 
    LEVEL_DEBUG    = 2 
    LEVEL_WARNNING = 4 
    LEVEL_ERROR    = 8 
    LEVEL_FATAL    = 16

    TRACE_INFO     = 32
    TRACE_DEBUG    = 64
    TRACE_WARNNING = 128 
    TRACE_ERROR    = 256 
    TRACE_FATAL    = 512 
)
```
