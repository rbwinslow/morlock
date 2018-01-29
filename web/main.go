package main

import (
	"net/http"
	"io/ioutil"
	"fmt"
	"os"
)

var (
	templates map[string]string = map[string]string{
		"html/index.html": "",
	}
)

func main() {
	for name := range templates {
		text, err := ioutil.ReadFile(name)
		if err != nil {
			exitf("Couldn't load template \"%s\"\n", name)
		}
		templates[name] = string(text)
	}

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/api/history", HistoryHandler)
	http.ListenAndServe(":8008", nil)
}

func exitf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
