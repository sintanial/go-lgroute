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
)

var stderr *log.Logger

func init() {
	stderr = log.New(os.Stderr, "", 0)
}


type Router struct {
	Key  []byte
	File *os.File
	flag int
}

func (r *Router) Write(data []byte) (n int, err error) {
	data = append(data, []byte("\r\n")...)
	return r.File.Write(data)
}

func (r *Router) Contains(s []byte) bool {
	return bytes.Contains(s, r.Key)
}

func (r *Router) String() string {
	var s string
	if r.flag == os.O_APPEND {
		s = ">>"
	} else {
		s = ">"
	}

	return "router: " + string(r.Key) + s + r.File.Name()
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

	file, err := os.OpenFile(params[1], os.O_WRONLY | os.O_CREATE | flag, 0644)
	if err != nil {
		return nil, err
	}

	return &Router{[]byte(params[0]), file, flag}, nil
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

func NewRouters(args []string) *Routers {
	var routers []*Router
	for _, arg := range args {
		router, err := NewRouter(arg)
		if err != nil {
			stderr.Println(err.Error())
			continue
		}
		routers = append(routers, router)
	}

	return &Routers{
		Routers: routers,
	}
}

func CanonicalArgs(args []string) []string {
	var result []string
	for _, arg := range args {
		if string(arg[0]) != "-" {
			result = append(result, arg)
		}
	}

	return result
}

func main() {
	inparallel := flag.Bool("p", false, "Run in parallel")
	flag.Parse()

	args := CanonicalArgs(os.Args[1:len(os.Args)])

	routers := NewRouters(args)
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
