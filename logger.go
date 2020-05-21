package glog

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	//"io"
	"github.com/slclub/goqueue"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

// default level. also is most commonly used.
var ALL_LEVEL = LEVEL_INFO + LEVEL_DEBUG + LEVEL_WARNNING + LEVEL_ERROR + LEVEL_FATAL + TRACE_ERROR + TRACE_FATAL

// debuging. = 1024 -1
var ALL_TRACE = ALL_LEVEL + TRACE_INFO + TRACE_DEBUG + TRACE_WARNNING

var level_name = map[int]string{
	LEVEL_INFO:     "INFO ",
	LEVEL_DEBUG:    "DEBUG",
	LEVEL_WARNNING: "WARN ",
	LEVEL_ERROR:    "ERROR",
	LEVEL_FATAL:    "FATAL",
}

var log_mgr = &logManager{}

func init() {
	log_mgr.show_time = true
	log_mgr.level_security = ALL_LEVEL
	log_mgr.pool_log.New = func() interface{} {
		return log_mgr.allocateLog()
	}
	ring := goqueue.NewQueue()
	//ring.Set()
	log_mgr.ring = ring

	// write now
	log_mgr.min_write = 1024
	log_mgr.trace_len = 5024
	log_mgr.write_now = make(chan byte, 2)
	log_mgr.tick_time = 10
	//log_mgr.time_format = "2020-05-19 00:00:00"
	log_mgr.time_format = "2006-01-02 15:04:05"
	log_mgr.buf = new(bytes.Buffer)
	log_mgr.log_file = createFileLog()

	// start writing file routine.
	go log_mgr.deamon()
}

//------------------------------------------------log manager ----------------------------------------------------
type logManager struct {
	to_stderr      bool      // log to os.stderr flag.
	to_both        bool      // to the file and os.stderr
	level_security int       // print level by level you defined. but dont include error and fatal level.
	trace_len      int       // set the trace bytes length.
	time_format    string    // log info time format
	pool_log       sync.Pool // cache loggin object . GET:pool.Get()(*logging) , pool.Put(c)
	show_time      bool      // check whether display time.
	log_file       *filelog
	log_file_err   *filelog
	ring           goqueue.Ring
	buf            *bytes.Buffer
	min_write      int
	write_now      chan byte     // write now.
	tick_time      time.Duration // write now.
}

func (m *logManager) allocateLog() *logging {
	return &logging{buf: new(bytes.Buffer)}
}

// level check
func (m *logManager) levelCheck(level int) bool {
	if m.level_security&level > 0 {
		return true
	}
	return false
}

// follow your level check the trace opend.
func (m *logManager) traceCheck(level int) bool {
	lt := level << 5
	if lt&m.level_security > 0 {
		return true
	}
	return false
}

func (m *logManager) levelSet(level int) {
	m.level_security |= level
}

func (m *logManager) getLevelName(level int) string {
	return level_name[level]
}

func (m *logManager) timeCheck() bool {
	return m.show_time
}

// set log name
func (m *logManager) SetLogName(name_pre string) {
	if name_pre == "" {
		return
	}
	m.log_file.prefix = name_pre
	if m.log_file_err != nil {
		m.log_file_err.prefix = name_pre + ".error"
	}
}

// set log relative path.
func (m *logManager) SetLogDir(log_dir string) {
	m.log_file.dir_log = log_dir
}

// set log file abs path.
func (m *logManager) SetLogAbs(log_dir string) {
	if log_dir == "" {
		return
	}
	m.log_file.dir_abs = log_dir

	if m.log_file_err != nil {
		m.log_file_err.dir_abs = log_dir
	}
}

func (m *logManager) SetPath(dir_abs string, dir_log string) {
	m.SetLogAbs(dir_abs)
	m.SetLogDir(dir_log)
}

func (m *logManager) getLog(level int) *filelog {
	if !m.levelCheck(level) {
		return nil
	}

	sl := level & (LEVEL_ERROR | LEVEL_ERROR)
	file := m.log_file
	if sl > 0 && m.log_file_err != nil {
		file = m.log_file_err
	}

	return file
}

// here support concurrency invoke.
func (m *logManager) output(level int, log []byte) {

	if !flag.Parsed() {
		os.Stderr.Write([]byte("ERROR:log before flag.parsed."))
		//os.Stderr.Write(log)
	}
	if m.to_stderr {
		os.Stderr.Write(log)
	}
	// write log in ring queue.
	m.ring.Write(log)
	// notice deamon routine now.
	m.write_now <- 1
}

// read log data from ring queue.
// this is rune on single routine.
func (m *logManager) Flush() (err error) {

	//select {
	//case finish_num := <-m.write_now:
	//	if m.ring.Size() < m.min_write && finish_num < 2 {
	//		return 0, nil
	//	}
	//	m.flushToFile()
	//default:
	//	m.flushToFile()
	//}
	//return
	m.flushToFile()
	return nil
}

func (m *logManager) flushToFile() (size int64) {

	m.buf.Write(m.ring.Read(0))
	size = int64(m.buf.Len())
	// TODO: separate the error log file. now no did it.
	m.getLog(LEVEL_INFO).Write(m.buf.Bytes())
	m.buf.Reset()

	//testcode
	//fmt.Println("[WRITE][FILE][OK]")
	return
}

// This deamon should only runing in one routine.
//
func (m *logManager) deamon() {
	//// 定义一个cron运行器
	//c := cron.New()
	//// 定时5秒，每5秒执行print5
	//c.AddFunc("*/5 * * * * *", print5)
	//// 定时15秒，每5秒执行print5
	//c.AddFunc("*/15 * * * * *", print15)

	//// 开始
	//c.Start()
	//defer c.Stop()

	// we choice another cron task method.
	tick1 := time.NewTicker(time.Second * m.tick_time)
	for {
		//fmt.Println("[DEAMON][FOR]", m.ring.Size())
		select {
		// listen write channel finis:=1
		case finish := <-m.write_now:
			//testcode
			//fmt.Println("[DEAMON][FROM WRITER]", m.ring.Size(), "[FINISH]", finish)

			if m.ring.Size() < m.min_write && finish < 2 {
				break
			}
			m.Flush()
			//testcode
			//fmt.Println("[DEAMON][FROM WRITER]", m.ring.Size())
		case <-tick1.C:
			// cron write data to file and not care data size.
			m.Flush()
			//tick1.Reset(time.Second * 1)
			//testcode
			//fmt.Println("[DEAMON][TICK]", m.ring.Size())

		}
	}
}

// create logging.
func (m *logManager) createLog(level int, method string, depth int, args ...interface{}) {
	log := m.pool_log.Get().(*logging)
	log.level = level

	switch method {
	case "print":
		log.print(args...)
	case "println":
		log.println(args...)
	case "printDepth":
		log.printDepth(depth, args...)
	}
	log.Reset()
	m.pool_log.Put(log)
}

// create logging.
func (m *logManager) createLogF(level int, format string, args ...interface{}) {
	log := m.pool_log.Get().(*logging)
	log.level = level

	log.printf(format, args...)
	m.pool_log.Put(log)
}

//--------------------------------------------logging method ---------------------------------------
type logging struct {
	buf   *bytes.Buffer
	level int // current log level.
}

func (l *logging) Reset() {
	l.level = 0
	l.buf.Reset()
}

func (l *logging) header(depth int) (*bytes.Buffer, string, int) {
	_, file, line, ok := runtime.Caller(5 + depth)
	if !ok {
		file = "???"
		line = -1
		return l.formatHeader(file, line), file, line
	}
	slash := strings.LastIndex(file, "/")
	if slash >= 0 {
		file = file[slash+1:]
	}

	return l.formatHeader(file, line), file, line
}

// format every log info header
func (l *logging) formatHeader(file string, line int) *bytes.Buffer {

	if log_mgr.timeCheck() {
		now := time.Now()
		hts := fmt.Sprintf(now.Format(log_mgr.time_format))
		l.buf.WriteString(hts)
		l.buf.WriteString(" ")
	}

	l.buf.WriteString(log_mgr.getLevelName(l.level))
	l.buf.WriteString(" ")

	//
	if log_mgr.traceCheck(l.level) {
		l.buf.WriteString(file)
		l.buf.WriteString(":")
		l.buf.WriteString(strconv.Itoa(line))
		l.buf.WriteString(" ")
	}

	return l.buf
}

func (l *logging) println(args ...interface{}) {
	fmt.Fprintln(l.buf, args...)
	l.printTrace()
	log_mgr.output(l.level, l.buf.Bytes())
}

func (l *logging) lineBreak() {
	if l.buf.Bytes()[l.buf.Len()-1] != '\n' {
		l.buf.WriteByte('\n')
	}
}

func (l *logging) printDepth(depth int, args ...interface{}) {
	l.header(depth)
	fmt.Fprint(l.buf, args...)
	l.lineBreak()
	l.printTrace()
	log_mgr.output(l.level, l.buf.Bytes())
}

func (l *logging) print(args ...interface{}) {
	l.printDepth(0, args...)
}

func (l *logging) printf(format string, args ...interface{}) {
	l.header(0)
	fmt.Fprintf(l.buf, format, args...)
	l.lineBreak()
	l.printTrace()
	log_mgr.output(l.level, l.buf.Bytes())
}

func (l *logging) printTrace() {
	if !log_mgr.traceCheck(l.level) {
		return
	}
	tmp_buf := stack(log_mgr.trace_len)
	//fmt.Fprint(l.buf, string(tmp_buf))
	//l.lineBreak()
	bytes.Trim(tmp_buf, "\x00")
	l.buf.Write(tmp_buf)
	//l.buf.WriteString(string(tmp_buf))
	//fmt.Println("STACK LEVEL CHECK", l.level, log_mgr.level_security, l.buf.Len(), len(tmp_buf))
}

// ---------------------------------------------------------------------------------------------------------------
// info
func Info(args ...interface{}) {
	log_mgr.createLog(LEVEL_INFO, "print", 0, args...)
}

func Infoln(args ...interface{}) {
	log_mgr.createLog(LEVEL_INFO, "println", 0, args...)
}

func Infof(format string, args ...interface{}) {
	log_mgr.createLogF(LEVEL_INFO, format, args...)
}
func InfoDepth(depth int, args ...interface{}) {
	log_mgr.createLog(LEVEL_INFO, "printDepth", depth, args...)
}

// debug
func Debug(args ...interface{}) {
	log_mgr.createLog(LEVEL_DEBUG, "print", 0, args...)
}

func Debugln(args ...interface{}) {
	log_mgr.createLog(LEVEL_DEBUG, "println", 0, args...)
}

func Debugf(format string, args ...interface{}) {
	log_mgr.createLogF(LEVEL_DEBUG, format, args...)
}
func DebugDepth(depth int, args ...interface{}) {
	log_mgr.createLog(LEVEL_DEBUG, "printDepth", depth, args...)
}

// warn
func Warnning(args ...interface{}) {
	log_mgr.createLog(LEVEL_WARNNING, "print", 0, args...)
}
func Warnningln(args ...interface{}) {
	log_mgr.createLog(LEVEL_WARNNING, "println", 0, args...)
}
func Warnningf(format string, args ...interface{}) {
	log_mgr.createLogF(LEVEL_WARNNING, format, args...)
}
func WarnningDepth(depth int, args ...interface{}) {
	log_mgr.createLog(LEVEL_WARNNING, "printDepth", depth, args...)
}

//error
func Error(args ...interface{}) {
	log_mgr.createLog(LEVEL_ERROR, "print", 0, args...)
}

func Errorln(args ...interface{}) {
	log_mgr.createLog(LEVEL_ERROR, "println", 0, args...)
}

func Errorf(format string, args ...interface{}) {
	log_mgr.createLogF(LEVEL_ERROR, format, args...)
}
func ErrorDepth(depth int, args ...interface{}) {
	log_mgr.createLog(LEVEL_ERROR, "printDepth", depth, args...)
}

// fatal
func Fatal(args ...interface{}) {
	log_mgr.createLog(LEVEL_FATAL, "print", 0, args...)
}

func Fatalln(args ...interface{}) {
	log_mgr.createLog(LEVEL_FATAL, "println", 0, args...)
}

func Fatalf(format string, args ...interface{}) {
	log_mgr.createLogF(LEVEL_FATAL, format, args...)
}
func FatalDepth(depth int, args ...interface{}) {
	log_mgr.createLog(LEVEL_FATAL, "printDepth", depth, args...)
}

// set log manager paramter.
func Set(field string, args ...interface{}) {
	if len(args) < 1 || args[0] == nil {
		fmt.Println("ERROR:Set log path need to at least one paramter ")
		return
	}
	field = strings.ToLower(field)
	switch field {
	// log file path
	case "path":
		if len(args) < 2 {
			fmt.Println("Set log path need to 2 paramter 1:abs path, 2:relative path")
			return
		}
		log_mgr.SetLogAbs(args[0].(string))
		log_mgr.SetLogDir(args[1].(string))
		fmt.Println("path print Set", log_mgr.log_file.fullpath())
	case "name":
		log_mgr.SetLogName(args[0].(string))
	case "tick":

		tick_time, ok := args[0].(time.Duration)
		if ok {
			log_mgr.tick_time = tick_time
		}
	case "head":
		log_mgr.log_file.head_create = args[0].(string)
	case "show_time":
		log_mgr.show_time = args[0].(bool)
	case "format":
		log_mgr.time_format = args[0].(string)
	case "debug":
		if check, ok := args[0].(bool); ok {
			if check {
				log_mgr.level_security = ALL_LEVEL
			} else {
				log_mgr.level_security = ALL_LEVEL - LEVEL_DEBUG
			}
			return
		}

		if level, ok := args[0].(int); ok {
			log_mgr.level_security = level
		}
	case "stderr":
		if check, ok := args[0].(bool); ok {
			log_mgr.to_stderr = check
		}
	}
}

// -----------------------------------------------trace location---------------------------------------------------
// log trace string error
var err_trace_syntax = errors.New("[log_syntax][trace_error][not valid trace string]")

//
////
//type traceLocation struct {
//	file string
//	line int
//}
//
//func (t *traceLocation) String() string {
//	return fmt.Sprintf("%s:%d", t.file, t.line)
//}
//
//func (t *traceLocation) Set(value string) error {
//	if value == "" {
//		t.line = 0
//		t.file = ""
//		return nil
//	}
//
//	fields := strings.Split(value, ":")
//	if len(fields) != 2 {
//		return err_trace_syntax
//	}
//
//	// file, line := fields[0], fields[1]
//	if !strings.Contains(fields[0], ".") {
//		return err_trace_syntax
//	}
//
//	v, err := strconv.Atoi(fields[1])
//	if err != nil {
//		return err_trace_syntax
//	}
//
//	if v < 0 {
//		return errors.New(err_trace_syntax.Error() + "line is zero")
//	}
//
//	t.file = fields[0]
//	t.line = v
//	return nil
//}

//

// utils function.
// invoke runtime stack.
func stack(size int) []byte {
	//var trace = make([]byte, 1024, size)
	//runtime.Stack(trace, true)
	trace := debug.Stack()
	return trace[600:]
}
