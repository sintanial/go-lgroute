package main

import (
	"os"
	"bufio"
	"strings"
	"errors"
	"log"
	"bytes"
	"io"
	"fmt"
	"flag"
	"regexp"
	"strconv"
	"runtime"
	"time"
)

const (
	BYTE = 1
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

var stderr *log.Logger

func init() {
	stderr = log.New(os.Stderr, "", 0)
}
// [subs]>>test.log!100mb
type Router struct {
	Key            []byte
	File           *os.File
	Bound          int
	Arg            string
	Compressor     string
	WatcherTimeout time.Duration
	watcher        *time.Timer
}

func (r *Router) Write(data []byte) (n int, err error) {
	data = append(data, []byte("\r\n")...)
	return r.File.Write(data)
}

func (r *Router) Contains(s []byte) bool {
	return bytes.Contains(s, r.Key)
}

func (r *Router) String() string {
	return "router: " + r.Arg
}

func (r *Router) RunWatcher() {
	if r.Bound > 0 {
		if r.watcher != nil {
			r.watcher.Stop()
		}

		r.watcher = time.AfterFunc(r.WatcherTimeout, func() {
			stat, err := r.File.Stat()
			if err != nil {
				// todo: корректно обработать
			} else if int(stat.Size()) >= r.Bound {
				r.Compress()
			}


			if r.watcher != nil {r.watcher.Reset(r.WatcherTimeout)}
		})
	}
}

func (r *Router) SetCompressor(algorithm string, timeout time.Duration) {
	r.Compressor = algorithm
	r.WatcherTimeout = timeout
}

func (r *Router) Compress() {
	// TODO: доделать
}

func NewRouter(arg string) (router *Router, err error) {
	var params []string
	var flag int
	if strings.Contains(arg, ">>") {
		params = strings.Split(arg, ">>")
		flag = os.O_APPEND
	} else if strings.Contains(arg, ">") {
		params = strings.Split(arg, ">")
		flag = os.O_TRUNC
	} else {
		return nil, errors.New(`invalid rediraction symbol in argument "` + arg + `"`)
	}


	if len(params) < 2 {
		return nil, errors.New(`invalid argument "` + arg + `"`)
	}

	fl := strings.Split(params[1], "!")

	filename := fl[0]
	bound := 0

	if len(fl) == 2 {
		bound = ToByte(fl[1])
	}

	file, err := os.OpenFile(filename, os.O_WRONLY | os.O_CREATE | flag, 0644)
	if err != nil {return nil, errors.New(`failed to open file, because: ` + err.Error())}

	return &Router{
		Key: []byte(params[0]),
		File: file,
		Arg: arg,
		Bound: bound,
	}, nil
}


type Routers struct {
	Routers    []*Router
	InParallel bool
}

func (r *Routers) handle(line []byte) {
	contained := false
	for _, router := range r.Routers {
		if router.Contains(line) {
			if _, err := router.Write(line); err != nil {
				stderr.Println(err.Error())
			} else {
				contained = true
			}
		}
	}

	if contained != true {
		fmt.Println(string(line))
	}
}

func (r *Routers) Handle(line []byte) {
	if r.InParallel {
		go r.handle(line)
	} else {
		r.handle(line)
	}
}

func NewRouters(args []string, compressor string, watchtime time.Duration) *Routers {
	var routers []*Router
	for _, arg := range args {
		router, err := NewRouter(arg)
		if err != nil {
			stderr.Println(err.Error())
			continue
		}

		router.SetCompressor(compressor, watchtime)
		router.RunWatcher()

		routers = append(routers, router)
	}

	return &Routers{
		Routers: routers,
	}
}

func CanonicalArgs(args []string) []string {
	var result []string
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if string(arg[0]) != "-" {
			result = append(result, arg)
		}
	}

	return result
}

var bytesRegex = regexp.MustCompile(`(\d+)(m|k|g)?`)
func ToByte(s string) int {
	res := bytesRegex.FindStringSubmatch(s)
	if len(res) < 3 {
		return 0
	}

	i, _ := strconv.Atoi(res[1])

	tp := res[2]

	switch tp {
	case "k": return i * KILOBYTE
	case "m": return i * MEGABYTE
	case "g": return i * GIGABYTE
	default: return i
	}

	return 0
}

func main() {
	inparallel := flag.Bool("p", false, "run in parallel")
	maxproc := flag.Int("m", -1, "max process")
	cmptm := flag.Int("t", 10, "compress file timeout check")
	cmpalg := flag.String("a", "gzip", "compress algorithm, allow gzip|flate")
	flag.Parse()

	if cmpalg != "gzip" || cmpalg != "flate" {
		stderr.Println("invalid compress algorithm")
		return
	}

	if *maxproc >= 1 {
		runtime.GOMAXPROCS(maxproc)
	}

	args := CanonicalArgs(os.Args[1:len(os.Args)])

	routers := NewRouters(args, *cmptm * time.Second, cmpalg)
	routers.InParallel = *inparallel

	bf := bufio.NewReader(os.Stdin)
	for {
		line, _, err := bf.ReadLine()

		if err != nil {
			if err != io.EOF {
				stderr.Println(err.Error())
			}
			break
		}

		routers.Handle(line)
	}
}
