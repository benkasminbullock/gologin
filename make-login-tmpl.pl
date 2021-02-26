#!/home/ben/software/install/bin/perl
use warnings;
use strict;
use utf8;
use FindBin '$Bin';
use HTML::Make::Page 'make_page';
use File::Slurper 'write_text';

make_login ();
make_error ();
make_show_users ();

exit;

sub make_show_users
{
    my ($html, $body) = make_page (
	title => "Users",
    );
    $body->push (make_nav ());
    $body->push ('h1', text => 'Users');
    my $table = $body->push ('table', class => 'show');
    $table->add_text (<<EOF);
<tr><th>User</th><th>Password</th></tr>
{{range .}}
<tr><td>{{.Login}}</td><td>{{.Pass}}</td></tr>
{{end}}
EOF
    write_text ("$Bin/tmpl/show-users.html", $html->text ());
}


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

sub make_login
{
    my $title = 'Go login/cookie test';
    my ($html, $body) = make_page (
	title => $title,
	lang => 'en',
    );
    $body->push (make_nav ());
    $body->push ('h1', text => $title);
    $body->add_text ("{{if .L}}\n");
    my $table = $body->push ('table');
    my $crow = $table->push ('tr');
    $crow->push ('th', text => 'Your current cookie:');
    $crow->push ('td', text => '{{.L.Cookie}}');
    my $lrow = $table->push ('tr');
    $lrow->push ('th', text => 'Your login:');
    $lrow->push ('td', text => '{{.L.Login}}');
    my $prow = $table->push ('tr');
    $prow->push ('th', text => 'Your password:');
    $prow->push ('td', text => '{{.L.Pass}}');
    $body->add_text ("{{end}}\n");
    my $form = <<EOF;
<h3>Log in</h3>
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
}

sub make_nav
{
    my $nav = HTML::Make->new ('div');
    $nav->push ('a', href => '/', text => 'Top page');
    $nav->push ('a', href => '?action=logout', text => 'Log out');
    $nav->push ('a', href => '?show=users', text => 'Show users');
    $nav->push ('a', href => '?show=logins', text => 'Show logins');
    $nav->push ('a', href => '?control=stop', text => '🛑 Stop server');
    return $nav;
}

sub make_logout
{
    my ($html, $body) = nav_page ('Log out');
    $body->add_text (<<EOF);
<form>
<input type='submit' value='Log out'>
</form>
EOF
}

sub nav_page
{
    my ($title) = @_;
    my ($html, $body) = make_page (title => $title);
    $body->push (make_nav ());
    return ($html, $body);
}
