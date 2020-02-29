package fileserver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

//Server wraps data used to start HTTP Server
type Server struct {
	Port     int
	Path     string
	Dotfiles bool
	router   *mux.Router
	files    []dynFile
}

type dynFile struct {
	URLPath string
	File    *os.File
}

func (fs *Server) routes() {
	fs.router.HandleFunc("/", rootHandler()).Methods("GET")
	//Handle File requests
	fs.router.HandleFunc("/file/{path}", fs.fileHandler()).Methods("GET", "HEAD")

	//Handle dynamic pages, can post aribtrary data
	fs.router.PathPrefix("/dyn/").HandlerFunc(fs.dynCreateHandler()).Methods("POST")

	//Return what is posted, put, or patched
	fs.router.PathPrefix("/echo").HandlerFunc(echoHandler()).Methods("POST", "PUT", "PATCH")

	fs.router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	//Give control to mux
	fs.router.Handle("/", fs.router)

}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Cannont find: %s", r.URL.Path)
	w.WriteHeader(404)
	fmt.Fprintf(w, "404 File Not Found!")

}

//Start HTTP FileServer
func (fs *Server) Start() {
	fs.router = mux.NewRouter()
	fs.routes()
	addr := fmt.Sprintf(":%d", fs.Port)
	srv := &http.Server{
		Handler:      fs.router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func rootHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome. You got served.")
	}
}
func (fs Server) fileHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		var filePath string
		if []rune(vars["path"])[len(vars["path"])-1] != '/' {
			filePath = fmt.Sprintf("%s/%s", fs.Path, vars["path"])

		} else {
			filePath = fmt.Sprintf("%s%s", fs.Path, vars["path"])

		}

		// Handle dotfiles and moving up the directory
		log.Println(filePath)
		if vars["path"][0] == '.' && !fs.Dotfiles {
			w.WriteHeader(404)
			fmt.Fprintf(w, "File Not Found\n")
			log.Printf("Attempt to access %s", filePath)
			return
		}

		file, err := os.Open(filePath)
		defer file.Close()
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "File Not Found\n")
			return
		}
		buf := make([]byte, 512)
		_, err = file.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		contentType := http.DetectContentType(buf)
		w.Header().Add("Content-Type", contentType)
		log.Println(contentType)

		fStat, _ := file.Stat()
		w.Header().Add("Content-Length", strconv.FormatInt(fStat.Size(), 10))

		var kbyte int64 = 1024
		var bufSize int64 = kbyte * 1024
		var bufNum int64 = 0
		fBuf := make([]byte, bufSize)
		bRead := int(bufSize)
		file.Seek(bufSize*bufNum, 0)
		for bRead == int(bufSize) {
			bRead, err = file.Read(fBuf)
			fmt.Printf("bytes read: %d\n", bRead)
			if bRead == int(bufSize) {
				w.Write(fBuf)

			} else {
				w.Write(fBuf[:bRead])
			}
			bufNum++
			file.Seek(bufSize*bufNum, 0)
		}

	}
}

func echoHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("path: %s", r.URL.Path)
		var arbJSON interface{}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(data, &arbJSON)
		json, _ := json.MarshalIndent(&arbJSON, "", "  ")
		fmt.Fprint(w, string(json))
		fmt.Fprint(w, "\n")

	}
}

func (f *dynFile) dynReadHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		f.File.Seek(0, 0)
		buf, err := ioutil.ReadAll(f.File)
		//file extension
		path := strings.Split(r.URL.Path, "/")
		filename := path[len(path)-1]
		path = strings.Split(filename, ".")
		ext := path[len(path)-1]

		switch ext {
		case "html":
			w.Header().Add("Content-Type", "text/html")
		case "json":
			w.Header().Add("Content-Type", "application/json")
		case "xml":
			w.Header().Add("Content-Type", "application/xml")
		case "pdf":
			w.Header().Add("Content-Type", "application/pdf")
		default:
			w.Header().Add("Content-Type", "text/plain")
		}

		if err != nil {
			log.Printf("Could not data for path %s", f.URLPath)
			w.WriteHeader(404)
			return
		}
		w.Write(buf)

	}
}
func (fs *Server) dynCreateHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		for _, f := range fs.files {
			if r.URL.Path == f.URLPath {
				w.WriteHeader(405)
				fmt.Fprintln(w, "Cannot POST to same URL twice")
				return
			}

		}

		tmpfile, err := ioutil.TempFile("", "ygs")
		if err != nil {
			log.Println("Could not open temp file, continuing...")
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		io.Copy(tmpfile, r.Body)

		file := dynFile{
			URLPath: r.URL.Path,
			File:    tmpfile,
		}
		fs.files = append(fs.files, file)

		fs.router.HandleFunc(r.URL.Path, file.dynReadHandler()).Methods("GET", "HEAD")
		fs.router.HandleFunc(r.URL.Path, file.dynUpdateHandler()).Methods("PUT")

		log.Printf("Created endpoint for path: %s", file.URLPath)

	}
}
func (f *dynFile) dynUpdateHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("dynUpdateHandler Called")

		if r.URL.Path == f.URLPath {
			tmpfile, err := ioutil.TempFile("", "ygs")
			if err != nil {
				log.Println("Could not open temp file, continuing...")
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			io.Copy(tmpfile, r.Body)
			os.Remove(f.File.Name())
			f.File = tmpfile
		}
	}
}

//TurnDownServer is used to clean up after process is asked to exit
func (fs *Server) TurnDownServer() {
	for _, file := range fs.files {
		log.Printf("Removing file: %s", file.File.Name())
		os.Remove(file.File.Name())
	}
}
