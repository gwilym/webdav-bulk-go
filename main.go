package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var directory_path string
var prefix_len int
var webdav_url string

func usage() {
	fmt.Fprintln(os.Stderr, "WebDAV bulk upload and download tool")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage for uploads:   ", os.Args[0], "[OPTION]... DIRECTORY URL")
	fmt.Fprintln(os.Stderr, "Usage for downloads: ", os.Args[0], "[OPTION]... URL DIRECTORY")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "OPTIONS (in alphabetal order)")
	fmt.Fprintln(os.Stderr, "        -d              Use Digest authentication")
	fmt.Fprintln(os.Stderr, "        -n NUMBER       Copy NUMBER files at the same time (default: 5)")
	fmt.Fprintln(os.Stderr, "        -p PASSWORD     Password to use for WebDAV")
	fmt.Fprintln(os.Stderr, "        -u USERNAME     Username to use for WebDAV")
	fmt.Fprintln(os.Stderr)
}

func visitDirectory(path string, f os.FileInfo, err error) error {
	relative_path := path[len(directory_path):]
	if relative_path != "" {
		fmt.Println("Visited", relative_path)
	}
	return nil
}

func main() {
	// var local = flag.String("local", "", "Local directory path")
	// var url = flag.String("host", "", "WebDAV URL")
	// var username = flag.String("username", "", "WebDAV Username")
	// var password = flag.String("password", "", "WebDAV Password")
	//

	threads := flag.Int("n", 5, "")
	tasks := make(chan *string, 64)

	flag.Usage = usage
	flag.Parse()

	directory_path = strings.TrimRight(flag.Arg(0), "/")
	prefix_len = len(directory_path) + 1
	webdav_url = flag.Arg(1)

	var wg sync.WaitGroup

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go func() {
			for foo := range tasks {
				fmt.Println("Shall process", directory_path, *foo)
			}
			wg.Done()
		}()
	}

	filepath.Walk(directory_path, func(path string, f os.FileInfo, err error) error {
		if len(path) > prefix_len {
			relative_path := path[prefix_len:]
			if relative_path != "" {
				tasks <- &relative_path
			}
		}
		return nil
	})
	close(tasks)
}
