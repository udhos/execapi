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
	"syscall"

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

type requestBody struct {
	Cmd []string `json:"cmd" yaml:"cmd"`
}

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

	var errExecMsg string

	cmd := exec.Command(reqBody.Cmd[0], reqBody.Cmd[1:]...)
	stdoutStderr, errExec := cmd.CombinedOutput()
	if errExec != nil {

		var exitStatus int

		if exitError, isExitError := errExec.(*exec.ExitError); isExitError {
			exitStatus = exitError.Sys().(syscall.WaitStatus).ExitStatus()
		}

		errExecMsg = fmt.Sprintf("%s: exec error: exit_status=%d: %v", me, exitStatus, errExec)
		log.Print(errExecMsg)
	}

	output := string(stdoutStderr)

	log.Printf("%s: output: %s", me, output)

	fmt.Fprintln(w, output)

	if errExec != nil {
		// show error
		fmt.Fprintln(w, errExecMsg)
	}
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
