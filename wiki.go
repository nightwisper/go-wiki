package main

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var templateCache = template.Must(template.ParseFiles("view.html", "edit.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

type Page struct {
	Title string
	Body  []byte
}

type Index struct {
	Pages []string
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filename)

	if body == nil && err != nil {
		return nil, err
	}

	return &Page{Title: title, Body: body}, nil
}

func getTitle(responseWriter http.ResponseWriter, request *http.Request) (string, error) {
	match := validPath.FindStringSubmatch(request.URL.Path)

	if match == nil {
		http.NotFound(responseWriter, request)
		return "", errors.New("Invalid Page Title")
	}

	return match[2], nil
}

func viewHandler(responseWriter http.ResponseWriter, request *http.Request, title string) {
	page, err := loadPage(title)

	if err != nil {
		http.Redirect(responseWriter, request, "/edit/"+title, http.StatusFound)
	}

	renderTemplate(responseWriter, "view", page)
}

func editHandler(responseWriter http.ResponseWriter, request *http.Request, title string) {
	page, err := loadPage(title)

	if err != nil {
		// new page
		page = &Page{Title: title}
	}

	renderTemplate(responseWriter, "edit", page)
}

func saveHandler(responseWriter http.ResponseWriter, request *http.Request, title string) {
	request.ParseForm()

	var body string

	for key, value := range request.Form {
		if key == "body" {
			body = strings.Join(value, "")
		}
	}

	filepath := title + ".txt"
	page, err := loadPage(filepath)

	if err != nil {
		page = &Page{Title: title, Body: []byte(body)}
	}

	err = page.save()

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(responseWriter, request, "/view/"+title, http.StatusFound)
}

func indexHandler(responseWriter http.ResponseWriter, request *http.Request) {
	files, err := os.ReadDir("./")

	if err != nil {
		log.Fatal(err)
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}

	pages := []string{}

	for _, file := range files {
		fileName := file.Name()

		if strings.Contains(fileName, ".txt") {
			pages = append(pages, strings.Replace(fileName, ".txt", "", 1))
		}
	}

	templ, err := template.ParseFiles("index.html")

	if err != nil {
		log.Fatal(err)
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}

	idx := &Index{Pages: pages}

	err = templ.Execute(responseWriter, idx)

	if err != nil {
		log.Fatal(err)
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}
}

func renderTemplate(respWriter http.ResponseWriter, tmpl string, page *Page) {
	err := templateCache.ExecuteTemplate(respWriter, tmpl+".html", page)
	if err != nil {
		log.Fatal(err)
		http.Error(respWriter, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		match := validPath.FindStringSubmatch(request.URL.Path)

		if match == nil {
			http.NotFound(responseWriter, request)
			return
		}

		fn(responseWriter, request, match[2])
	}
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		log.Printf("%s %s %s\n", request.RemoteAddr, request.Method, request.URL)
		handler.ServeHTTP(responseWriter, request)
	})
}

func main() {
	log.SetPrefix("Wiki: ")
	log.SetFlags(0)

	http.HandleFunc("/index", indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", logRequest(http.DefaultServeMux)))
}
