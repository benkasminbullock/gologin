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
	user2logins map[string][]login
}

func (l *logintest) errorPage(err error) {
	fmt.Fprintf(l.w, "Error: %s", err)
}

// Write a new login file.
func (l *logintest) writeLogin() {
	bytes, err := json.MarshalIndent(l.logins, "", "\t")
	if err != nil {
		l.errorPage(err)
		log.Print(err)
	}
	err = ioutil.WriteFile(l.loginfile, bytes, 0666)
	if err != nil {
		l.errorPage(err)
		log.Print(err)
	}
}

// Append a login to the login file.
func (l *logintest) appendLogin() {
	l.writeLogin()
}

// Store "li" to the login file.
func (l *logintest) storeLogin(li login) {
	l.logins = append(l.logins, li)
	if _, err := os.Stat(l.loginfile); os.IsNotExist(err) {
		l.writeLogin()
		return
	}
	l.appendLogin()
}

var cookieName = "gologin"
var cookiePath = "/"

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

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
	if found {
		log.Printf("Found cookie %s", v)
		if offset > 0 {
			l.logins = append(l.logins[0:offset-1], l.logins[offset+1:]...)
		} else {
			l.logins = l.logins[1:]
		}

		l.writeLogin()
		return
	}
	log.Printf("Did not find cookie %s", v)
}

func handler(l *logintest) {
	l.L = login{}
	cookie, err := l.r.Cookie(cookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			l.errorPage(err)
			log.Fatalf("Error from .Cookie: %s", err)
		}
	}
	l.L.Login = l.r.FormValue("user-name")
	if len(l.L.Login) > 0 {
		if cookie != nil {
			l.DeleteOldCookie(cookie)
		}
		l.L.Pass = l.r.FormValue("password")
		l.L.Cookie = randSeq(5)
		l.storeLogin(l.L)
		l.SetCookie(l.L.Cookie)
	}
	err = l.templates.Execute(l.w, l)
	if err != nil {
		l.errorPage(err)
		log.Fatalf("Error executing templates: %s", err)
	}
	// Blank out the login values.
	l.L = login{}
}

func MakeHandler(l *logintest, f func(*logintest)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l.w = w
		l.r = r
		f(l)
	}
}

func (l *logintest) readConfigJSON(file string) {
	dfile := l.dir + "/" + file
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

func (l *logintest) ReadUsers() {
	dfile := l.dir + "/" + "users.json"
	b, err := ioutil.ReadFile(dfile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("There is no users file '%s'", dfile)
			// Allow empty user file
			return
		}
		log.Fatalf("Error reading user file %s: %s", dfile, err)
	}
	json.Unmarshal(b, &l.users)
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
}

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
	l.loginfile = l.dir + "/" + "logins.json"
	l.ReadLogins()
}

func main() {
	var l logintest
	l.setup()
	http.HandleFunc("/", MakeHandler(&l, handler))
	err := http.ListenAndServe(":9191", nil)
	if err != nil {
		log.Fatal(err)
	}
}
