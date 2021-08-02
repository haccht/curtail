package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
)

func listen(addr string) (net.Listener, error) {
	if strings.Contains(addr, "/") {
		return net.Listen("unix", addr)
	}

	return net.Listen("tcp", addr)
}

func serve(addr string, mux *http.ServeMux) error {
	conn, err := listen(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-sigCh

		conn.Close()
		os.Exit(1)
	}()

	return http.Serve(conn, mux)
}

func tailFunc(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if !strings.Contains(strings.ToLower(r.Header.Get("User-Agent")), "curl") {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var rs []*regexp.Regexp
	for _, v := range query["q"] {
		re, err := regexp.Compile(v)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		rs = append(rs, re)
	}

	lines := make(chan string)
	for _, filepath := range flag.Args() {
		go func(filepath string) {
			config := tail.Config{
				Follow:   true,
				ReOpen:   true,
				Logger:   tail.DiscardingLogger,
				Location: &tail.SeekInfo{0, os.SEEK_END},
			}

			t, err := tail.TailFile(filepath, config)
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}

			for line := range t.Lines {
				lines <- line.Text
			}

			err = t.Wait()
			if err != nil {
				fmt.Fprintln(w, err)
			}
		}(filepath)
	}

	for {
		select {
		case line := <-lines:
			matched := true
			for _, re := range rs {
				matched = matched && re.MatchString(line)
			}
			if !matched {
				continue
			}
			fmt.Fprintln(w, line)
		case <-ticker.C:
			// send NUL to keep the connection alive
			fmt.Fprint(w, "\x00")
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func main() {
	addr := flag.String("addr", ":8080", "Address to listen on")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Target file must be specified.")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", tailFunc)

	fmt.Printf("Listening on %s\n", *addr)
	log.Fatal(serve(*addr, mux))
}
