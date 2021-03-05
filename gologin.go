// Example of making persistent cookies in Go.

package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type LoginStore interface {
	CheckPassword(user string, password string) (found bool)
	FindUser(user string) (found bool)
	LookUpCookie(cookie string) (user string, found bool, err error)
	DeleteCookie(cookie string) (err error)
	Init(dir string) (err error)
	StoreLogin(user string, cookie string) (err error)
	DeleteAllLogins() (err error)
	// These methods are for display
	Login(name string, cookie string) (user interface{})
	Users() (users interface{})
	Logins() (logins interface{})
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
	store  LoginStore
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
	e := struct {
		Error string
	}{
		Error: fmt.Sprintf(format, a...),
	}
	errortmpl := l.templates.Lookup("error.html")
	err := errortmpl.Execute(l.w, e)
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

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Make a string to be used as a cookie
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (l *logintest) SetCookie(encoded string) {
	cookie := http.Cookie{
		Name:  cookieName,
		Value: encoded,
		Path:  cookiePath,
	}
	http.SetCookie(l.w, &cookie)
}

func (l *logintest) clearCookie() {
	cookie := http.Cookie{
		Name:    cookieName,
		Value:   "",
		Path:    cookiePath,
		MaxAge:  -1,
		Expires: time.Now().Add(-100 * time.Hour),
	}
	http.SetCookie(l.w, &cookie)
}

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

func (l *logintest) HandleLogin(name string, cookie *http.Cookie) (newcookie string, ok bool) {
	ok = l.store.FindUser(name)
	if !ok {
		l.errorPage("Unknown user '%s'", name)
		return "", false
	}
	if cookie != nil {
		l.message("Deleting old cookie %s", cookie.Value)
		l.store.DeleteCookie(cookie.Value)
	}
	pass := l.r.FormValue("password")
	if !l.store.CheckPassword(name, pass) {
		l.errorPage("Wrong password '%s' for %s", pass, name)
		return "", false
	}
	newcookie = randSeq(5)
	l.message("Password %s correct for %s, setting cookie to %s",
		pass, name, newcookie)
	l.SetCookie(newcookie)
	l.storeLogin(name, newcookie)
	return newcookie, true
}

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
	}
}

func (l *logintest) DoTemplate(template string, thing interface{}) {
	tmpl := l.templates.Lookup(template)
	if tmpl == nil {
		panic(fmt.Sprintf("No such template %s", template))
	}
	err := tmpl.Execute(l.w, thing)
	if err != nil {
		panic(fmt.Sprintf("Error from tmpl.Execute: %s", err))
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
		cookie, err := l.r.Cookie(cookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				l.errorPage("You were not logged in")
				return
			}
			l.errorPage("Error from r.Cookie: %s", err)
		}
		l.store.DeleteCookie(cookie.Value)
		l.clearCookie()
		l.errorPage("You are now logged out")
		return
	}
	if action == "delete-all" {
		l.message("Deleting all logins")
		l.store.DeleteAllLogins()
		l.errorPage("All current logins have been deleted")
		return
	}
	l.errorPage("Unknown action '%s'", action)
}

// Handle web requests
func handler(l *logintest) {
	url := l.r.URL
	if strings.Contains(url.Path, "favicon.ico") {
		l.w.WriteHeader(http.StatusNotFound)
		return
	}
	control := l.r.FormValue("control")
	if len(control) > 0 {
		l.HandleControl(control)
		return
	}
	cookie, err := l.r.Cookie(cookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			l.errorPage("Error from cookie: %s", err)
		}
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
	logtmp := l.templates.Lookup("login.html")
	cookieOk := false
	if cookie != nil {
		l.message("Cookie %s", cookie.Value)
	}
	var name string
	var cookieval string
	if cookie != nil {
		cookieval = cookie.Value
		if len(cookieval) > 0 {
			name, cookieOk = l.LookUpCookie(cookieval)
		}
	}
	if !cookieOk {
		if cookie == nil {
			l.message("No cookie was sent")
		} else {
			l.message("Cookie %s was not found", cookieval)
			l.clearCookie()
		}
		loginOk := false
		name = l.r.FormValue("user-name")
		if len(name) > 0 {
			l.message("Looking for user name %s", name)
			cookieval, loginOk = l.HandleLogin(name, cookie)
			if !loginOk {
				return
			}
		}
	}
	err = logtmp.Execute(l.w, l.store.Login(name, cookieval))
	if err != nil {
		l.errorPage("Error executing template: %s", err)
		return
	}
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
	l.store = &store.Store{}
	l.store.Init(l.dir)
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
