SRCS=gologin.go store/store.go login/login.go

all: gologin tmpl/login.html users.json

gologin: $(SRCS)
	go build -o $@ gologin.go

tmpl/login.html: make-login-tmpl.pl
	./make-login-tmpl.pl

users.json: make-users.pl
	./make-users.pl

clean:
	rm -f \
	gologin \
	tmpl/login.html \
	logins.json \
	users.json \
	zzzz
	purge -r
