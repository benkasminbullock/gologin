// Example of making persistent cookies in Go.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gologin/login"
	"gologin/store"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Login interface {
	Init() (err error)
	LogIn(w http.ResponseWriter, r *http.Request, user string, password string) (err error)
	LogOut(w http.ResponseWriter, r *http.Request) (err error)
	User(w http.ResponseWriter, r *http.Request) (user string, err error)
}

type logintest struct {
	w         http.ResponseWriter
	r         *http.Request
	templates *template.Template
	dir       string
	config    map[string]string
	// True to print messages.
	verbose bool
	// True if we are serving.
	serving bool

	server *http.Server
	store  store.Store
	login  login.Login
	User   string
	Thing  interface{}
}

//  __  __
// |  \/  | ___  ___ ___  __ _  __ _  ___  ___
// | |\/| |/ _ \/ __/ __|/ _` |/ _` |/ _ \/ __|
// | |  | |  __/\__ \__ \ (_| | (_| |  __/\__ \
// |_|  |_|\___||___/___/\__,_|\__, |\___||___/
//                             |___/

func (l *logintest) message(format string, a ...interface{}) {
	if !l.verbose {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	_, file = filepath.Split(file)
	fmt.Printf("%s:%d: ", file, line)
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

/* Send an error message to the browser. */
func (l *logintest) errorPage(format string, a ...interface{}) {
	l.message(format, a...)
	if !l.serving {
		return
	}
	e := map[string]string{"Error": fmt.Sprintf(format, a...)}
	l.Thing = e
	errortmpl := l.templates.Lookup("error.html")
	err := errortmpl.Execute(l.w, l)
	if err != nil {
		log.Printf("Error executing error template: %s", err)
	}
}

/* Send an error message to the browser and quit. */
func (l *logintest) Fatalf(format string, a ...interface{}) {
	if l.serving {
		l.errorPage(format, a)
	}
	log.Fatalf(format, a)
}

//  ____  _
// / ___|| |_ ___  _ __ __ _  __ _  ___
// \___ \| __/ _ \| '__/ _` |/ _` |/ _ \
//  ___) | || (_) | | | (_| | (_| |  __/
// |____/ \__\___/|_|  \__,_|\__, |\___|
//                           |___/

// Store "li" to the login file.
func (l *logintest) storeLogin(user string, cookie string) {
	err := l.store.StoreLogin(user, cookie)
	if err != nil {
		l.errorPage("Error storing cookie for %s: %s", user, err)
	}
}

//   ____            _    _
//  / ___|___   ___ | | _(_) ___  ___
// | |   / _ \ / _ \| |/ / |/ _ \/ __|
// | |__| (_) | (_) |   <| |  __/\__ \
//  \____\___/ \___/|_|\_\_|\___||___/
//

var cookieName = "gologin"
var cookiePath = "/"

// Read a file from the local directory
func (l *logintest) ReadFile(file string) (b []byte) {
	dfile := filepath.Join(l.dir, file)
	b, err := ioutil.ReadFile(dfile)
	if err != nil {
		if os.IsNotExist(err) {
			l.message("There is no file '%s'", dfile)
			// Allow empty user file
			return b
		}
		l.Fatalf("Error reading file %s: %s", dfile, err)
	}
	return b
}

//  ____
// |  _ \ __ _  __ _  ___  ___
// | |_) / _` |/ _` |/ _ \/ __|
// |  __/ (_| | (_| |  __/\__ \
// |_|   \__,_|\__, |\___||___/
//             |___/

// https://medium.com/@int128/shutdown-http-server-by-endpoint-in-go-2a0e2d7f9b8c
func (l *logintest) StopServing() {
	l.message("Stopping server")
	ctx, cancel := context.WithCancel(l.r.Context())
	defer cancel()
	l.errorPage("Stopping server")
	go func() {
		if err := l.server.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()
	l.message("Server has stopped")
	l.serving = false
}

func (l *logintest) HandleControl(control string) {
	l.message("Received control message '%s'", control)
	if control == "stop" {
		l.StopServing()
		return
	}
	l.errorPage("Unknown control message '%s'", control)
}

func (l *logintest) DoTemplate(template string, thing interface{}) {
	tmpl := l.templates.Lookup(template)
	if tmpl == nil {
		panic(fmt.Sprintf("No such template %s", template))
	}
	l.Thing = thing
	err := tmpl.Execute(l.w, l)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from tmpl.Execute: %s\n", err)
	}
}

// Handle showing something
func (l *logintest) HandleShow(show string) {
	if show == "logins" {
		l.DoTemplate("show-logins.html", l.store.Logins())
	} else if show == "users" {
		l.DoTemplate("show-users.html", l.store.Users())
	}
}

func (l *logintest) LookUpCookie(cookie string) (login string, ok bool) {
	login, ok, err := l.store.LookUpCookie(cookie)
	if err != nil {
		l.errorPage("Error looking up cookie: %s", err)
		return login, false
	}
	return login, ok
}

func (l *logintest) HandleAction(action string) {
	if action == "logout" {
		l.login.LogOut(l.w, l.r)
		l.errorPage("You are now logged out")
		return
	}
	if action == "delete-all" {
		l.message("Deleting all logins")
		l.store.DeleteAllLogins()
		l.errorPage("All current logins have been deleted")
		return
	}
	if action == "login" {
		l.LoginPage()
		return
	}
	l.errorPage("Unknown action '%s'", action)
}

func (l *logintest) LoginPage() {
	user := l.r.FormValue("user-name")
	if len(user) > 0 {
		password := l.r.FormValue("password")
		if len(password) == 0 {
			l.errorPage("No password")
			return
		}
		err := l.login.LogIn(l.w, l.r, user, password)
		if err != nil {
			l.errorPage("Error logging in: %s", err)
			return
		}
		http.Redirect(l.w, l.r, "/", http.StatusFound)
	}
	l.DoTemplate("login.html", map[string]string{})
}

// Handle web requests
func handler(l *logintest) {
	url := l.r.URL
	if strings.Contains(url.Path, "favicon.ico") {
		l.w.WriteHeader(http.StatusNotFound)
		return
	}
	var err error
	l.User, err = l.login.User(l.w, l.r)
	if err != nil {
		l.errorPage("Error getting user: %s", err)
		return
	}
	control := l.r.FormValue("control")
	if len(control) > 0 {
		l.HandleControl(control)
		return
	}
	show := l.r.FormValue("show")
	if len(show) > 0 {
		l.HandleShow(show)
		return
	}
	action := l.r.FormValue("action")
	if len(action) > 0 {
		l.HandleAction(action)
		return
	}
	l.DoTemplate("top.html", map[string]string{})
}

func MakeHandler(l *logintest, f func(*logintest)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l.w = w
		l.r = r
		f(l)
	}
}

// Read the configuration file from the local directory
func (l *logintest) readConfigJSON(file string) {
	b := l.ReadFile(file)
	l.config = make(map[string]string)
	err := json.Unmarshal(b, &l.config)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %s", err)
	}
}

// Set up "l" to serve web pages.
func (l *logintest) setup() {
	self := os.Args[0]
	var err error
	l.dir, err = filepath.Abs(filepath.Dir(self))
	if err != nil {
		log.Fatalf("Error getting directory of %s: %s",
			self, err)
	}
	l.readConfigJSON("config.txt")
	l.templates, err = template.ParseGlob(l.dir + "/tmpl/*.html")
	if err != nil {
		log.Fatalf("Error reading templates: %s", err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	l := logintest{
		verbose: true,
	}
	l.setup()
	l.store = store.Store{}
	l.store.Init(l.dir)
	l.login.Init(&l.store, cookieName, cookiePath)
	serve := ":" + l.config["port"]
	l.server = &http.Server{Addr: serve}
	http.HandleFunc("/", MakeHandler(&l, handler))
	l.message("Serving on %s", serve)
	l.serving = true
	err := l.server.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			log.Fatalf("Error from server: %s", err)
		}
	}
	l.serving = false
}
