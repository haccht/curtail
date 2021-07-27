package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/hpcloud/tail"
)

func main() {
	addr := flag.String("addr", ":8080", "TCP address to listen on")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("Filepath must be specified.")
	}

	filepath := flag.Arg(0)
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

		q := r.URL.Query()

		var rs []*regexp.Regexp
		for _, v := range q["q"] {
			re, err := regexp.Compile(v)
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}
			rs = append(rs, re)
		}

		t, err := tail.TailFile(filepath, tailconf)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		for line := range t.Lines {
			pass := true
			for _, re := range rs {
				pass = pass && re.MatchString(line.Text)
			}
			if !pass {
				continue
			}

			fmt.Fprintln(w, line.Text)
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
