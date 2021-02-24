// Example of making persistent cookies in Go.

package main

import (
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
}

/* Send an error message to the browser. */
func (l *logintest) errorPage(format string, a ...interface{}) {
	l.message(format, a...)
	if !l.serving {
		return
	}
	fmt.Fprintf(l.w, "<div class='error'>\n")
	fmt.Fprintf(l.w, format, a...)
	fmt.Fprintf(l.w, "</div>\n")
}

// Write the logins from l.logins into the login file.
func (l *logintest) WriteLogins() {
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

var cookieName = "gologin"
var cookiePath = "/"

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Make a string to be used as a cookie
func randSeq(n int) string {
	//	rand.Seed(res.now.UnixNano())
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

func handler(l *logintest) {
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
		user, ok := l.name2user[name]
		if !ok {
			l.errorPage("Unknown user '%s'", name)
			return
		}
		if cookie != nil {
			l.DeleteOldCookie(cookie)
		}
		l.L.Login = name
		pass := l.r.FormValue("password")
		if pass != user.Pass {
			l.errorPage("Wrong password '%s' should be '%s'",
				pass, user.Pass)
			return
		}
		l.L.Pass = pass
		l.L.Cookie = randSeq(5)
		l.storeLogin(l.L)
		l.SetCookie(l.L.Cookie)
	}
	err = l.templates.Execute(l.w, l)
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

func (l *logintest) readConfigJSON(file string) {
	dfile := filepath.Join(l.dir, file)
	b, err := ioutil.ReadFile(dfile)
	if err != nil {
		log.Fatalf("Error reading %s: %s", dfile, err)
	}
	l.config = make(map[string]string)
	err = json.Unmarshal(b, &l.config)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %s", err)
	}
}

func (l *logintest) Fatalf(format string, a ...interface{}) {
	if l.serving {
		l.errorPage(format, a)
	}
	log.Fatalf(format, a)
}

func (l *logintest) ReadFile(file string) (b []byte) {
	dfile := filepath.Join(l.dir, "users.json")
	b, err := ioutil.ReadFile(dfile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("There is no users file '%s'", dfile)
			// Allow empty user file
			return
		}
		l.Fatalf("Error reading user file %s: %s", dfile, err)
	}
	return b
}

func (l *logintest) ReadUsers() {
	b := l.ReadFile("users.json")
	json.Unmarshal(b, &l.users)
	l.name2user = make(map[string]*user, len(l.users))
	for i, r := range l.users {
		l.name2user[r.Login] = &l.users[i]
	}
}

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
	l.templates, err = template.ParseFiles(l.dir + "/tmpl/login.html")
	if err != nil {
		log.Fatalf("Error reading templates: %s", err)
	}
	l.ReadUsers()
	l.loginfile = filepath.Join(l.dir, "logins.json")
	l.ReadLogins()
}

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

func main() {
	l := logintest{
		verbose: true,
	}
	l.setup()
	http.HandleFunc("/", MakeHandler(&l, handler))
	serve := ":" + l.config["port"]
	l.message("Serving on %s", serve)
	l.serving = true
	err := http.ListenAndServe(serve, nil)
	if err != nil {
		log.Fatalf("Error from server: %s", err)
	}
	l.serving = false
}
