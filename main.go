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

	filename := flag.Arg(0)
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

		t, err := tail.TailFile(filename, tailconf)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		for {
			select {
			case line := <-t.Lines:
				matched := true
				for _, re := range rs {
					matched = matched && re.MatchString(line.Text)
				}
				if !matched {
					continue
				}
				fmt.Fprintln(w, line.Text)
			case <-ticker.C:
				// send NUL to keep the connection alive
				fmt.Fprint(w, "\x00")
			}

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

		err = t.Wait()
		if err != nil {
			fmt.Fprintln(w, err)
		}
	})

	fmt.Printf("Listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
