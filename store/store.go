package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	//	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	//	"strings"
	"time"
)

type login struct {
	Login  string    `json:"login"`
	Pass   string    `json:"pass"`
	Cookie string    `json:"cookie"`
	Last   time.Time `json:"last"`
}

type user struct {
	Login string `json:"login"`
	Pass  string `json:"pass"`
}

type Store struct {
	// The directory where the files are stored
	dir string
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
}

func (s *Store) message(format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	_, file = filepath.Split(file)
	fmt.Printf("%s:%d: ", file, line)
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

// Read a file from the local directory
func (s *Store) readFile(file string) (b []byte, err error) {
	dfile := filepath.Join(s.dir, file)
	b, err = ioutil.ReadFile(dfile)
	if err != nil {
		if os.IsNotExist(err) {
			s.message("There is no file '%s'", dfile)
			// Allow empty user file
			return b, nil
		}
		return b, fmt.Errorf("Error reading file %s: %s", dfile, err)
	}
	return b, nil
}

// Write the logins from l.logins into the login file.
func (s *Store) writeLogins() (err error) {
	s.message("Writing logins file")
	bytes, err := json.MarshalIndent(s.logins, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.loginfile, bytes, 0666)
}

func (s *Store) initlogins() (err error) {
	s.cookie2user = make(map[string]*user, len(s.logins))
	s.user2logins = make(map[string][]*login, len(s.logins))
	for i, r := range s.logins {
		login := r.Login
		user := s.name2user[login]
		if user == nil {
			return fmt.Errorf("Can't find user with name '%s'", login)
		}
		s.cookie2user[r.Cookie] = user
		s.AddUserLogin(user.Login, &s.logins[i])
	}
	return nil
}

// Add a login to the user's list of logins.

func (s *Store) AddUserLogin(username string, lo *login) {
	uls := s.user2logins[username]
	uls = append(uls, lo)
	s.user2logins[username] = uls
}

func (s *Store) readLogins() (err error) {
	b, err := ioutil.ReadFile(s.loginfile)
	if err != nil {
		if os.IsNotExist(err) {
			s.message("There is no logins file '%s'", s.loginfile)
			// Allow empty user file
			s.initlogins()
			return
		}
		return err
	}
	err = json.Unmarshal(b, &s.logins)
	if err != nil {
		return err
	}
	return s.initlogins()
}

func (s *Store) Init(dir string) (err error) {
	s.dir = dir
	err = s.readLogins()
	if err != nil {
		return err
	}
	return s.readUsers()
}

// Read the file of users from the local directory
func (s *Store) readUsers() (err error) {
	b, err := s.readFile("users.json")
	if err != nil {
		return err
	}
	json.Unmarshal(b, &s.users)
	s.name2user = make(map[string]*user, len(s.users))
	for i, r := range s.users {
		s.name2user[r.Login] = &s.users[i]
	}
	return nil
}
func (s *Store) StoreLogin(user string, cookie string) (err error) {
	li := login{
		Login:  user,
		Cookie: cookie,
	}
	s.logins = append(s.logins, li)
	return nil
}

func (s *Store) CookieToLogin(cookie string) (found bool, user string, err error) {
	u, found := s.cookie2user[cookie]
	if !found {
		return false, "", nil
	}
	return true, u.Login, nil
}
