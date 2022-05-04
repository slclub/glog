package glog

/**
 * I used the queueRing. read in single go routine is safe.
 * there is no need to use lock.
 */
import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"io/ioutil"
	//"runtime"
	//"syscall"
)

var unit_size = map[string]int{
	"K": 1024,
	"M": 1024 * 1024,
	"G": 1024 * 1024 * 1024,
}
var zero_time time.Time

type filelog struct {
	dir_log     string
	dir_abs     string
	app_name    string
	app_path    string
	prefix      string
	suffix      string
	t           time.Time // create file time
	size        int64
	rotate_size int64 // =0: rotated rule is by date, >0 : rule is by size
	file        *os.File
	file_id     int
	head_create string
	keep_days	int64
}

func createFileLog() *filelog {
	path, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	fl := &filelog{
		app_name:    filepath.Base(os.Args[0]),
		app_path:    path,
		size:        0,
		t:           zero_time,
		file_id:     0,
		rotate_size: 0,
		head_create: "",
		keep_days:	30,  // 默认保留30天; Default keep the logs file alive of tirth days;
	}



	//fl.rotate(time.Now())
	return fl
}

func (fl *filelog) logName(t time.Time) string {
	if "" == fl.suffix {
		fl.suffix = "log"
	}
	if fl.prefix == "" {
		fl.prefix = fl.app_name
	}
	name := ""
	if t.IsZero() {
		name = fl.prefix
	} else {
		name = fmt.Sprintf("%s.%04d.%02d.%02d", fl.prefix, t.Year(), t.Month(), t.Day())
	}
	if fl.file_id != 0 {
		name = fmt.Sprintf("%s.%d", name, fl.file_id)
	}
	name += "." + fl.suffix
	return name
}

// create log floder.
func (fl *filelog) createLogDir() {

	full_path := fl.fullpath()
	fmt.Println("path print create log dir:", full_path)
	if ok, _ := isFileExist(full_path); ok {
		//fmt.Println("path exist create error:", full_path)
		return
	}
	err := os.MkdirAll(full_path, os.ModePerm)
	if err != nil {
		fmt.Println("MKDIR:", err)
	}
	return
}

func (fl *filelog) fullpath() string {
	if fl.dir_abs == "" {
		fl.dir_abs = fl.app_path
	}
	full_path := fl.dir_abs + "/" + fl.dir_log
	full_path = strings.Replace(full_path, "//", "/", 5)
	return full_path
}

// follow time.Time get a name
func (fl *filelog) fullname(nt time.Time) string {
	full_path := fl.fullpath()
	return filepath.Join(full_path, fl.logName(nt))
}

var once sync.Once

func (fl *filelog) create_file(t time.Time) (fn *os.File, fname string, err error) {
	fl.createLogDir()
	name := fl.logName(t)
	fname = filepath.Join(fl.fullpath(), name)
	fn, err = os.Create(fname)
	return
}

func (fl *filelog) setRotateSize(size_str string) error {
	s1 := size_str[:len(size_str)-2]
	s2 := size_str[len(size_str)-1:]
	tmp, err := strconv.Atoi(strings.Replace(s1, " ", "", -1))
	if err != nil {
		return err
	}
	s2 = strings.ToUpper(s2)
	unit := unit_size[s2]
	fl.rotate_size = int64(tmp * unit)
	return nil
}

func (fl *filelog) reset() {
	fl.size = 0
}

func (fl *filelog) Close() {
	fl.file.Close()
}

func (fl *filelog) Sync() {
	fl.file.Sync()
}

// write data to file
func (fl *filelog) Write(p []byte) (size int, err error) {

	size = len(p)
	if size == 0 {
		return 0, nil
	}
	err = fl.rotate(time.Now())
	if err != nil {
		return 0, err
	}
	//testcode
	//fmt.Println(" file.Write size:", fl.size)

	fl.file.Write(p)
	fl.size += int64(size)
	return
}

// move file
func (fl *filelog) movefile() error {
	fullname := fl.fullname(fl.t)
	if ok, _ := isFileExist(fullname); ok {
		fl.file_id++
	}
	err := os.Rename(fl.fullname(zero_time), fl.fullname(fl.t))
	return err
}

func (fl *filelog) Flush() error {
	return nil
}

func (fl *filelog) rotate(t time.Time) (err error) {

	if !fl.checkRotate(t) {
		return nil
	}
	// testcode
	//fmt.Println("rotate:", fl.fullpath(), fl.fullname(time.Now()))
	now_t := time.Now()
	// When we start project, value of fl.t is zero time
	if isSameDate(fl.t, zero_time) {
		fl.t = getFileModifyTime(fl.fullname(zero_time))
	}
	//fmt.Println("[ROTATE][LOG][PATH]", fl.fullpath(), "[LOG_NAME]", fl.fullname(zero_time), "[MODIFY]",isSameDate(fl.t, time.Now()))

	if fl.file != nil || !isSameDate(fl.t, now_t ) {
		fl.Flush()
		fl.file.Close()
		fl.movefile()
		//fmt.Println("[REMOVE][LOG][PATH]", fl.fullpath(), "[LOG_NAME]", fl.fullname(zero_time), "[MODIFY]",isSameDate(fl.t, time.Now()) )
	}
	fname := fl.fullname(zero_time)
	//fl.t = getFileModifyTime(fl.fullpath())
	if ok, _ := isFileExist(fname); ok && isSameDate(fl.t, time.Now()) {
		fl.file, err = os.OpenFile(fname, os.O_WRONLY|os.O_APPEND, 0666)
		fmt.Println("[OPEN_FILE][OK]", fname, ";", err)
		if err != nil {
			fmt.Println("[OPEN_FILE][ERROR]", fname, ";", err)
			panic("Log file maybe permission deny!")
		}
		//
		fl.reset()
		return nil
	}
	// create new log file
	fl.file, _, err = fl.create_file(zero_time)
	if err != nil {
		return err
	}
	//
	fl.reset()
	fl.t = time.Now()

	fl.file.Write([]byte(fl.head_create))
	go fl.dynamicDelete(fl.fullpath() )
	return nil
}

// check create a new log file.
func (fl *filelog) checkRotate(nt time.Time) bool {
	if fl.file == nil {
		return true
	}
	if fl.rotate_size > fl.size {
		return true
	}

	if isSameDate(fl.t, nt) {
		return false
	}
	return true
}

func (fl *filelog) reload() {
}

// delete old log files that were expired
func (fl *filelog) dynamicDelete(readdir string) {
	rdir , _ := ioutil.ReadDir(readdir)
	expire_time := time.Now().Unix() - fl.keep_days * 86400
	for _, fi := range rdir {
		full_file_name := filepath.Join(fl.fullpath(), fi.Name())
		if fi.IsDir() {
			fl.dynamicDelete( full_file_name )
		}
		if ok, _ := isFileExist(full_file_name ); ok  && getFileModifyTime( full_file_name ).Unix() < expire_time {
			os.Remove(full_file_name )
		}
	}
}

// --------------------------------------util function.-----------------------------
func isFileExist(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	//我这里判断了如果是0也算不存在
	if fileInfo.Size() == 0 {
		return false, nil
	}

	return false, err
}

func timespecToTime(ts int64) time.Time {
	return time.Unix(ts, 0)
}

func getFileModifyTime(file_path string) time.Time{
	finfo, err := os.Stat(file_path)
	if err != nil {
		fmt.Println("file is not exist:", file_path)
		return time.Unix(0, 0)
	}
	return finfo.ModTime()
	//sys_type := runtime.GOOS
	//fmt.Println("os type:", sys_type, finfo)
	//switch sys_type {
	//case "windows":
	//	stat_t := finfo.Sys().(*syscall.Win32FileAttributeData)
	//	//fmt.Println(timespecToTime(stat_t.LastAccessTime.Nanoseconds()/1e9))
	//	//fmt.Println(timespecToTime(stat_t.CreationTime.Nanoseconds()/1e9))
	//	//fmt.Println(timespecToTime(stat_t.LastWriteTime.Nanoseconds()/1e9))
	//	return timespecToTime(stat_t.CreationTime.Nanoseconds()/1e9)
	//case "linux":
	//	stat_t := finfo.Sys().(*syscall.Stat_t)
	//	//fmt.Println("文件创建时间", SecondToTime(linuxFileAttr.Ctim.Sec))
	//	//fmt.Println("最后访问时间", SecondToTime(linuxFileAttr.Atim.Sec))
	//	//fmt.Println("最后修改时间", SecondToTime(linuxFileAttr.Mtim.Sec))
	//	return timespecToTime(stat_t.Ctim.Sec)
	//case "darwin": //mac 时间
	//}
	//return timespecToTime(0)
}

func isSameDate(t time.Time, nt time.Time) bool {
	if t.Day() != nt.Day() {
		return false
	}

	if t.Month() != nt.Month() {
		return false
	}

	if t.Year() != nt.Year() {
		return false
	}
	return true
}
