// Example of making persistent cookies in Go.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type login struct {
	Login  string `json:"login"`
	Pass   string `json:"pass"`
	Cookie string `json:"cookie"`
}

type user struct {
	Login string `json:"login"`
	Pass  string `json:"pass"`
}

type logintest struct {
	w         http.ResponseWriter
	r         *http.Request
	templates *template.Template
	dir       string
	config    map[string]string
	L         login
	// The file containing the logins.
	loginfile string
	// List of all users.
	users []user
	// List of all logins.
	logins []login
	// Given a cookie, look up its user and log them in.
	cookie2user map[string]*user
	// Given a user, look up their logins.
	user2logins map[string][]*login
	name2user   map[string]*user
	// True to print messages.
	verbose bool
	// True if we are serving.
	serving bool

	server *http.Server
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
	errortmpl.Execute(l.w, e)
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

// Write the logins from l.logins into the login file.
func (l *logintest) WriteLogins() {
	l.message("Writing logins file")
	bytes, err := json.MarshalIndent(l.logins, "", "\t")
	if err != nil {
		l.errorPage("Error marshalling JSON: %s", err)
		return
	}
	err = ioutil.WriteFile(l.loginfile, bytes, 0666)
	if err != nil {
		l.errorPage("Error writing %s: %s", l.loginfile, err)
		return
	}
}

// Append a login to the login file.
func (l *logintest) appendLogin() {
	l.WriteLogins()
}

// Store "li" to the login file.
func (l *logintest) storeLogin(li login) {
	l.logins = append(l.logins, li)
	if _, err := os.Stat(l.loginfile); os.IsNotExist(err) {
		l.WriteLogins()
		return
	}
	l.appendLogin()
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

func (l *logintest) DeleteOldCookie(cookie *http.Cookie) {
	v := cookie.Value
	l.message("Looking for a cookie with value '%s'", v)
	l.ReadLogins()
	found := false
	offset := -1
	for i := range l.logins {
		if l.logins[i].Cookie == v {
			found = true
			offset = i
			break
		}
	}
	if !found {
		l.message("Did not find the cookie")
		return
	}
	l.message("Found cookie %s", v)
	l.logins = append(l.logins[0:offset], l.logins[offset+1:]...)
	l.WriteLogins()
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

// Read the file of users from the local directory
func (l *logintest) ReadUsers() {
	b := l.ReadFile("users.json")
	json.Unmarshal(b, &l.users)
	l.name2user = make(map[string]*user, len(l.users))
	for i, r := range l.users {
		l.name2user[r.Login] = &l.users[i]
	}
}

// Read the file of logins (user+cookie) from the local directory
func (l *logintest) ReadLogins() {
	b, err := ioutil.ReadFile(l.loginfile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("There is no logins file '%s'", l.loginfile)
			// Allow empty user file
			return
		}
		log.Fatalf("Error reading user file %s: %s", l.loginfile, err)
	}
	json.Unmarshal(b, &l.logins)
	l.cookie2user = make(map[string]*user, len(l.logins))
	l.user2logins = make(map[string][]*login, len(l.users))
	for i, r := range l.logins {
		login := r.Login
		user := l.name2user[login]
		if user == nil {
			l.errorPage("Can't find user with name '%s'", login)
			continue
		}
		l.cookie2user[r.Cookie] = user
		l.user2logins[login] = append(l.user2logins[login], &l.logins[i])
	}
}

//  ____
// |  _ \ __ _  __ _  ___  ___
// | |_) / _` |/ _` |/ _ \/ __|
// |  __/ (_| | (_| |  __/\__ \
// |_|   \__,_|\__, |\___||___/
//             |___/

func (l *logintest) HandleLogin(name string, cookie *http.Cookie) bool {
	user, ok := l.name2user[name]
	if !ok {
		l.errorPage("Unknown user '%s'", name)
		return false
	}
	if cookie != nil {
		l.DeleteOldCookie(cookie)
	}
	l.L.Login = name
	pass := l.r.FormValue("password")
	if pass != user.Pass {
		l.errorPage("Wrong password '%s' should be '%s'",
			pass, user.Pass)
		return false
	}
	l.L.Pass = pass
	l.L.Cookie = randSeq(5)
	l.storeLogin(l.L)
	l.SetCookie(l.L.Cookie)
	return true
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
		fmt.Fprintf(l.w, "<html><body><table>\n")
		for i, r := range l.logins {
			fmt.Fprintf(l.w, "<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				i, r.Login, r.Pass, r.Cookie)
		}
		fmt.Fprintf(l.w, "</table></body></html>\n")
	} else if show == "users" {
		l.DoTemplate("show-users.html", l.users)
	}
}

func (l *logintest) LookUpCookie(cookie string) {
	user, ok := l.cookie2user[cookie]
	if !ok {
		l.message("Cookie %s not found", cookie)
		return
	}
	l.L.Login = user.Login
	l.L.Pass = user.Pass
	l.L.Cookie = cookie
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
		l.DeleteOldCookie(cookie)
		l.errorPage("You are now logged out")
		return
	}
	l.errorPage("Unknown action '%s'", action)
}

// Handle web requests
func handler(l *logintest) {
	control := l.r.FormValue("control")
	if len(control) > 0 {
		l.HandleControl(control)
		return
	}
	l.L = login{}
	defer func() {
		// Blank out the login values.
		l.L = login{}
	}()
	cookie, err := l.r.Cookie(cookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			l.errorPage("Error from cookie: %s", err)
		}
	}
	name := l.r.FormValue("user-name")
	if len(name) > 0 {
		if !l.HandleLogin(name, cookie) {
			return
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
	if len(cookie.Value) > 0 {
		l.LookUpCookie(cookie.Value)
	}
	logtmp := l.templates.Lookup("login.html")
	err = logtmp.Execute(l.w, l)
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
	l.ReadUsers()
	l.loginfile = filepath.Join(l.dir, "logins.json")
	l.ReadLogins()
}

func main() {
	l := logintest{
		verbose: true,
	}
	l.setup()
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
