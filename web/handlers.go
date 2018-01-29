package main

import (
	"net/http"
	"fmt"
	"html/template"
	"github.com/rbwinslow/morlock/api"
)

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm() ; err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := r.Form.Get("path")
	repo, err := api.OpenLocalGitRepo(p, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fileSubPath := p[len(repo.Path)+1:]
	var commits api.CommitList
	out, err := repo.History(fileSubPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for c := range out {
		commits = append(commits, c)
	}
	js, err := commits.ToJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintln(w, string(js))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := newHtmlTemplate("index")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "index", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newHtmlTemplate(name string) (*template.Template, error) {
	tmpl := template.New(name)
	tmpl = tmpl.Delims("[[", "]]")
	return tmpl.Parse(templates[fmt.Sprintf("html/%s.html", name)])
}
