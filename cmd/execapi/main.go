// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

const version = "0.1.0"

func getVersion(me string) string {
	return fmt.Sprintf("%s version=%s runtime=%s GOOS=%s GOARCH=%s GOMAXPROCS=%d",
		me, version, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
}

type config struct {
}

func main() {

	app := config{}

	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	{
		v := getVersion(me)
		if showVersion {
			fmt.Println(v)
			return
		}
		log.Print(v)
	}

	addr := envString("ADDR", ":8080")
	route := envString("ROUTE", "/exec")
	health := envString("HEALTH", "/health")

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	const root = "/"

	register(mux, addr, root, func(w http.ResponseWriter, r *http.Request) { handlerRoot(&app, w, r) })
	register(mux, addr, health, func(w http.ResponseWriter, r *http.Request) { handlerHealth(&app, w, r) })
	register(mux, addr, route, func(w http.ResponseWriter, r *http.Request) { handlerPath(&app, w, r) })

	go listenAndServe(server, addr)

	<-chan struct{}(nil)
}

func register(mux *http.ServeMux, addr, path string, handler http.HandlerFunc) {
	mux.HandleFunc(path, handler)
	log.Printf("registered on port %s path %s", addr, path)
}

func listenAndServe(s *http.Server, addr string) {
	log.Printf("listening on port %s", addr)
	err := s.ListenAndServe()
	log.Printf("listening on port %s: %v", addr, err)
}

/*
// httpJSON replies to the request with the specified error message and HTTP code.
// It does not otherwise end the request; the caller should ensure no further
// writes are done to w.
// The error message should be JSON.
func httpJSON(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, error)
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("toJSON: %v", err)
	}
	return string(b)
}
*/

type requestBody struct {
	Cmd []string `json:"cmd" yaml:"cmd"`
}

/*
func response(app *config, w http.ResponseWriter, r *http.Request, status int, message string) {
	const me = "response"

	hostname, errHost := os.Hostname()
	if errHost != nil {
		log.Printf("%s hostname error: %v", me, errHost)
	}

	// take a copy of the body
	reqBody, errRead := io.ReadAll(r.Body)
	if errRead != nil {
		log.Printf("%s: body read error: %v", me, errRead)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // restore it

	errForm := r.ParseForm()
	if errForm != nil {
		log.Printf("%s: form error: %v", me, errForm)
	}

	errMultipart := r.ParseMultipartForm(32 << 20)
	if errMultipart != nil {
		log.Printf("%s: form multipart error: %v", me, errMultipart)
	}

	params := map[string]string{}

	for _, p := range app.paramList {
		params[p] = r.FormValue(p)
	}

	reply := responseBody{
		Request: responseRequest{
			Headers:   r.Header,
			Method:    r.Method,
			URI:       r.RequestURI,
			Host:      r.Host,
			Body:      string(reqBody),
			FormQuery: r.Form,
			FormPost:  r.PostForm,
			Params:    params,
		},
		Message:        message,
		Status:         status,
		ServerHostname: hostname,
		ServerVersion:  version,
	}

	body := toJSON(reply)

	httpJSON(w, body, status)
}
*/

func handlerRoot(app *config, w http.ResponseWriter, r *http.Request) {
	const me = "handlerRoot"
	log.Printf("%s: %s %s %s - 404 not found",
		me, r.RemoteAddr, r.Method, r.RequestURI)
	http.Error(w, "not found", 404)
}

func handlerHealth(app *config, w http.ResponseWriter, r *http.Request) {
	const me = "handlerHealth"
	log.Printf("%s: %s %s %s - 200 health ok",
		me, r.RemoteAddr, r.Method, r.RequestURI)
	fmt.Fprintln(w, "health ok")
}

func handlerPath(app *config, w http.ResponseWriter, r *http.Request) {
	const me = "handlerPath"

	body, errRead := io.ReadAll(r.Body)
	if errRead != nil {
		msg := fmt.Sprintf("%s: body read error: %v", me, errRead)
		log.Print(msg)
		http.Error(w, msg, 400)
		return
	}

	log.Printf("%s: %s %s %s - request: %v",
		me, r.RemoteAddr, r.Method, r.RequestURI, string(body))

	var reqBody requestBody

	errYaml := yaml.Unmarshal(body, &reqBody)
	if errYaml != nil {
		msg := fmt.Sprintf("%s: body yaml error: %v", me, errYaml)
		log.Print(msg)
		http.Error(w, msg, 400)
		return
	}

	if len(reqBody.Cmd) < 1 {
		msg := fmt.Sprintf("%s: missing command in request", me)
		log.Print(msg)
		http.Error(w, msg, 400)
		return
	}

	cmd := exec.Command(reqBody.Cmd[0], reqBody.Cmd[1:]...)
	stdoutStderr, errExec := cmd.CombinedOutput()
	if errExec != nil {
		msg := fmt.Sprintf("%s: exec error: %v", me, errExec)
		log.Print(msg)
		http.Error(w, msg, 500)
		return
	}

	output := string(stdoutStderr)

	log.Printf("%s: output: %s", me, output)

	fmt.Fprintln(w, output)
}

// envString extracts string from env var.
// It returns the provided defaultValue if the env var is empty.
// The string returned is also recorded in logs.
func envString(name string, defaultValue string) string {
	str := os.Getenv(name)
	if str != "" {
		log.Printf("%s=[%s] using %s=%s default=%s", name, str, name, str, defaultValue)
		return str
	}
	log.Printf("%s=[%s] using %s=%s default=%s", name, str, name, defaultValue, defaultValue)
	return defaultValue
}
