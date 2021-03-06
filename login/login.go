package login

import (
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"runtime"
	"time"
)

type LoginStore interface {
	CheckPassword(user string, password string) (found bool)
	FindUser(user string) (found bool)
	LookUpCookie(cookie string) (user string, found bool, err error)
	DeleteCookie(cookie string) (err error)
	Init(dir string) (err error)
	StoreLogin(user string, cookie string) (err error)
}

type Login struct {
	store      LoginStore
	cookieName string
	cookiePath string
}

func (lo *Login) message(format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	_, file = filepath.Split(file)
	fmt.Printf("%s:%d: ", file, line)
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

func (lo *Login) Init(store LoginStore, cookieName string, cookiePath string) (err error) {
	lo.store = store
	lo.cookieName = cookieName
	lo.cookiePath = cookiePath
	return err
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Make a string to be used as a cookie
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (lo *Login) clearCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:    lo.cookieName,
		Value:   "",
		Path:    lo.cookiePath,
		MaxAge:  -1,
		Expires: time.Now().Add(-100 * time.Hour),
	}
	http.SetCookie(w, &cookie)
}

func (lo *Login) setCookie(w http.ResponseWriter, encoded string) {
	cookie := http.Cookie{
		Name:  lo.cookieName,
		Value: encoded,
		Path:  lo.cookiePath,
	}
	http.SetCookie(w, &cookie)
}

func (lo *Login) LogIn(w http.ResponseWriter, r *http.Request, user string, password string) (err error) {
	ok := lo.store.FindUser(user)
	if !ok {
		return fmt.Errorf("Unknown user '%s'", user)
	}
	cookie, err := r.Cookie(lo.cookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			return err
		}
	}
	if cookie != nil {
		lo.message("Deleting old cookie %s", cookie.Value)
		lo.store.DeleteCookie(cookie.Value)
	}
	pwok := lo.store.CheckPassword(user, password)
	if !pwok {
		if cookie != nil {
			lo.clearCookie(w)
		}
		return fmt.Errorf("Wrong password for %s", user)
	}
	newcookie := randSeq(5)
	lo.message("Password %s correct for %s, setting cookie to %s",
		password, user, newcookie)
	lo.setCookie(w, newcookie)
	err = lo.store.StoreLogin(user, newcookie)

	return err
}

func (lo *Login) LogOut(w http.ResponseWriter, r *http.Request) (err error) {
	cookie, err := r.Cookie(lo.cookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			lo.message("You were not logged in")
			return nil
		}
		return fmt.Errorf("Error from r.Cookie: %s", err)
	}
	fmt.Printf("%v\n", lo.store)
	lo.store.DeleteCookie(cookie.Value)
	lo.clearCookie(w)
	return err
}

func (lo *Login) User(w http.ResponseWriter, r *http.Request) (user string, err error) {
	cookie, err := r.Cookie(lo.cookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", err
	}
	if cookie == nil {
		return "", nil
	}
	login, ok, err := lo.store.LookUpCookie(cookie.Value)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	return login, nil
}
