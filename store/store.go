package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
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
	s.message("The login file is '%s'", s.loginfile)
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
	s.loginfile = filepath.Join(s.dir, "logins.json")
	err = s.readUsers()
	if err != nil {
		return err
	}
	return s.readLogins()
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
	s.user2logins[user] = append(s.user2logins[user], &s.logins[len(s.logins)-1])
	s.cookie2user[cookie] = s.name2user[user]
	return s.writeLogins()
}

func (s *Store) LookUpCookie(cookie string) (user string, found bool, err error) {
	u, found := s.cookie2user[cookie]
	if !found {
		return "", false, nil
	}
	return u.Login, true, nil
}

func (s *Store) DeleteCookie(cookie string) (err error) {
	s.message("Looking for a cookie with value '%s'", cookie)
	s.readLogins()
	found := false
	offset := -1
	for i := range s.logins {
		if s.logins[i].Cookie == cookie {
			found = true
			offset = i
			break
		}
	}
	if !found {
		s.message("Did not find the cookie")
		return err
	}
	s.message("Found cookie %s", cookie)
	s.logins = append(s.logins[0:offset], s.logins[offset+1:]...)
	return s.writeLogins()
}
func (s *Store) CheckPassword(login string, password string) (found bool) {
	user, ok := s.name2user[login]
	if !ok {
		return false
	}
	return password == user.Pass
}

func (s *Store) DeleteAllLogins() (err error) {
	err = os.Remove(s.loginfile)
	if err != nil {
		return err
	}
	s.logins = nil
	s.initlogins()
	return nil
}
func (s *Store) FindUser(name string) (found bool) {
	s.message("Looking for user %s", name)
	_, found = s.name2user[name]
	return found
}

func (s *Store) Users() (users interface{}) {
	return s.users
}
func (s *Store) Login(name string, cookie string) (u interface{}) {
	logins := s.user2logins[name]
	for i, r := range logins {
		if r.Cookie == cookie {
			return &logins[i]
		}
	}
	return &login{}
}
func (s *Store) Logins() (logins interface{}) {
	return s.logins
}
