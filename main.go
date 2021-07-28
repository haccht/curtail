package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hpcloud/tail"
)

func main() {
	addr := flag.String("addr", ":8080", "TCP address to listen on")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Target file must be specified.")
	}

	tailconf := tail.Config{
		Follow:   true,
		ReOpen:   true,
		Logger:   tail.DiscardingLogger,
		Location: &tail.SeekInfo{0, os.SEEK_END},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("User-Agent")), "curl") {
			return
		}

		query := r.URL.Query()
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
				t, err := tail.TailFile(filepath, tailconf)
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
	})

	fmt.Printf("Listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
