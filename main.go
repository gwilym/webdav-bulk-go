package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const TaskDirectionUp = 1
const TaskDirectionDown = 2

var directory_path string
var prefix_len int
var webdav_url string
var digest_auth *Authorization
var nonce_counter uint64
var tasks chan *task

type task struct {
	path      string
	direction int
}

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

func processTask(directory_path string, task *task) {
	file, err := os.Open(directory_path + "/" + task.path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	url := webdav_url + "/" + task.path

	if stat.IsDir() {
		// MKCOL
		// TODO need to ensure collections are created first
		// TODO adjust bc server side so this isn't required
		req, err := http.NewRequest("MKCOL", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		if digest_auth != nil {
			nc := int(atomic.AddUint64(&nonce_counter, 1))
			ApplyDigestAuth(req, digest_auth, nc)
		}

		log.Println("Creating", task.path)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		log.Println(task.path, resp.Status)
	} else {
		// PUT

		pr, pw := io.Pipe()
		bufin := bufio.NewReader(file)

		go func() {
			bufin.WriteTo(pw)
			pw.Close()
		}()

		req, err := http.NewRequest("PUT", url, pr)
		if err != nil {
			log.Fatal(err)
		}

		if digest_auth != nil {
			nc := int(atomic.AddUint64(&nonce_counter, 1))
			ApplyDigestAuth(req, digest_auth, nc)
		}

		log.Println("Sending", task.path)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			log.Fatal(task.path, " failed with status ", resp.Status)
		}
	}
}

func main() {
	// var local = flag.String("local", "", "Local directory path")
	// var url = flag.String("host", "", "WebDAV URL")
	// var username = flag.String("username", "", "WebDAV Username")
	// var password = flag.String("password", "", "WebDAV Password")

	threads := flag.Int("n", 5, "")
	tasks = make(chan *task, *threads+10)

	flag.Usage = usage
	flag.Parse()

	directory_path = strings.TrimRight(flag.Arg(0), "/")
	prefix_len = len(directory_path) + 1
	webdav_url = strings.TrimRight(flag.Arg(1), "/")
	task_direction := TaskDirectionUp

	parsed_url, err := url.Parse(webdav_url)
	if err != nil {
		log.Fatal(err)
	}

	if !(parsed_url.Scheme == "http" || parsed_url.Scheme == "https") {
		log.Fatal("Invalid url")
	}

	log.Println("Connecting to", parsed_url.Host, "...")

	resp, err := http.Head(webdav_url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 && resp.Header["Www-Authenticate"] != nil {
		nonce_counter = 1
		username := parsed_url.User.Username()
		log.Println("Logging in as", username, "...")
		password, _ := parsed_url.User.Password()
		parsed_url.User = nil
		digest_auth = GetAuthorization(username, password, resp)
		digest_req, err := http.NewRequest("HEAD", parsed_url.String(), nil)
		if err != nil {
			log.Fatal(err)
		}

		ApplyDigestAuth(digest_req, digest_auth, int(nonce_counter))
		digest_resp, err := http.DefaultClient.Do(digest_req)
		if err != nil {
			log.Fatal(err)
		}
		defer digest_resp.Body.Close()

		if digest_resp.StatusCode != 200 {
			log.Fatal("Login failed with status ", digest_resp.Status)
		}
		webdav_url = parsed_url.String()
	} else if resp.StatusCode != 200 {
		log.Fatal("Connection failed with status ", resp.Status)
	}

	var wg sync.WaitGroup

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go func() {
			for task := range tasks {
				processTask(directory_path, task)
			}
			wg.Done()
		}()
	}

	filepath.Walk(directory_path, func(path string, f os.FileInfo, err error) error {
		if len(path) > prefix_len {
			if f.IsDir() {
				relative_path := path[prefix_len:]
				tasks <- &task{path: relative_path, direction: task_direction}
			}
		}
		return nil
	})

	// TODO the process may have to wait until collections exist otherwise if
	//      the requests below could commence too early

	filepath.Walk(directory_path, func(path string, f os.FileInfo, err error) error {
		if len(path) > prefix_len {
			if !f.IsDir() {
				relative_path := path[prefix_len:]
				tasks <- &task{path: relative_path, direction: task_direction}
			}
		}
		return nil
	})

	close(tasks)
}
