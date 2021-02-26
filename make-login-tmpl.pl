#!/home/ben/software/install/bin/perl
use warnings;
use strict;
use utf8;
use FindBin '$Bin';
use HTML::Make::Page 'make_page';
use File::Slurper 'write_text';

my $title = 'Go login/cookie test';
my ($html, $body) = make_page (
    title => $title,
    lang => 'en',
);
my $nav = make_nav ();
$body->push ($nav);
$body->push ('h1', text => $title);
$body->add_text ("{{if .L}}\n");
my $table = $body->push ('table');
my $lrow = $table->push ('tr');
$lrow->push ('th', text => 'login');
$lrow->push ('td', text => '{{.L.Login}}');
my $prow = $table->push ('tr');
$prow->push ('th', text => 'pass');
$prow->push ('td', text => '{{.L.Pass}}');
my $crow = $table->push ('tr');
$crow->push ('th', text => 'cookie');
$crow->push ('td', text => '{{.L.Cookie}}');
$body->add_text ("{{end}}\n");
my $form = <<EOF;
<form id='login-form' method='POST'>
<b>Name:</b><input name='user-name' value="{{.L.Login}}">
<br>
<b>Password:</b><input name='password' value="{{.L.Pass}}">
<br>
<input type='submit'>
</form>
EOF
$body->add_text ($form);
write_text ("$Bin/tmpl/login.html", $html->text ());

make_error ();

exit;

sub make_error
{
    my $error = "{{.Error}}";
    my ($html, $body) = make_page (
	title => $error,
    );
    $body->push (make_nav ());
    $body->push ('div', class => 'error', text => $error);
    write_text ("$Bin/tmpl/error.html", $html->text ());
}

sub make_nav
{
    my $nav = HTML::Make->new ('div');
    $nav->push ('a', href => '?control=stop', text => 'Stop server');
    $nav->push ('a', href => '?show=users', text => 'Show users');
    $nav->push ('a', href => '?show=logins', text => 'Show logins');
    return $nav;
}
