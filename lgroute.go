package main

import (
	"os"
	"bufio"
	"strings"
	"errors"
	"log"
	"bytes"
	"io"
)

type RouterKey struct {
	Key  []byte
	File *os.File
}

func (rk *RouterKey) Write(data []byte) (n int, err error) {
	return rk.File.Write(data)
}

func (rk *RouterKey) Contains(s []byte) bool {
	return bytes.Contains(s, rk.Key)
}

func (rk *RouterKey) ContainsAndWrite(s []byte) (err error) {
	if rk.Contains(s) {
		s = append(s, []byte("\r\n")...)
		_, err = rk.Write(s)
	}

	return err
}

func ParseRouter(arg string) (router *RouterKey, err error) {
	var data []string
	var flag int
	if strings.Contains(arg, ">>") {
		data = strings.Split(arg, ">>")
		flag = os.O_APPEND
	} else if strings.Contains(arg, ">") {
		data = strings.Split(arg, ">")
		flag = os.O_TRUNC
	}


	if len(data) < 2 {
		return nil, errors.New(`invalid argument format "` + arg + `"`)
	}

	file, err := os.OpenFile(data[1], os.O_WRONLY | os.O_CREATE | flag, 0644)
	if err != nil {
		return nil, err
	}

	return &RouterKey{[]byte(data[0]), file}, nil
}


func main() {
	args := os.Args[1:len(os.Args)]

	stderr := log.New(os.Stderr, "", 0)

	keys := make([]*RouterKey, 0)
	for _, arg := range args {
		router, err := ParseRouter(arg)
		if err != nil {
			stderr.Println(err, router)
			continue
		}
		keys = append(keys, router)
	}

	bf := bufio.NewReader(os.Stdin)
	for {
		line, _, err := bf.ReadLine()

		for _, router := range keys {
			if err := router.ContainsAndWrite(line); err != nil {
				stderr.Println(err)
			}
		}

		if err != nil {
			if err != io.EOF {
				stderr.Println(err)
			}
			break
		}
	}
}
