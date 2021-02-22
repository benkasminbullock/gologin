SRCS=gologin.go

all: gologin tmpl/login.html

gologin: $(SRCS)
	go build -o $@ $(SRCS)

tmpl/login.html: make-login-tmpl.pl
	./make-login-tmpl.pl

clean:
	rm -f \
	gologin \
	tmpl/login.html \
	zzzz
	purge -r
