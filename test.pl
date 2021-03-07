#!/home/ben/software/install/bin/perl

# Test the functioning of gologin.

use warnings;
use strict;
use utf8;
use FindBin '$Bin';
use Test::More;
use JSON::Parse 'read_json';
use LWP::UserAgent;
use HTTP::CookieJar::LWP;
my $bin = "$Bin/gologin";
if (! -f $bin) {
    system ("make clean;make");
}
my $pid = fork ();
if ($pid == 0) {
    system ($bin);
#    print "Finished serving.\n";
    exit;
}
sleep (1);
#print "Not serving $pid.\n";
my $config = read_json ("$Bin/config.txt");
my $port = $config->{port};
my $cj = HTTP::CookieJar::LWP->new ();
my $ua = LWP::UserAgent->new (
    cookie_jar => $cj,
);
my $url = "http://localhost:$port";
my $got = $ua->get ($url);
ok ($got->is_success (), "Got $url");
#use Data::Dumper;
#print Dumper ($got);
my $login = ['user-name' => 'mariko', password => 'nyan'];
my $reply = $ua->post ("$url?action=login", $login);
ok ($reply->is_redirect (), "OK post req with password");
my @cookies = $cj->cookies_for ($url);
ok (@cookies, "Got cookies");
ok (scalar (@cookies) == 1, "Only one cookie");
is ($cookies[0]{name}, 'gologin', "Right cookie");
my $value = $cookies[0]{value};
ok (cookie_file ($value), "Found $value in cookie file");
$ua->get ("$url?action=logout");
@cookies = $cj->cookies_for ($url);
ok (scalar (@cookies) == 0, "Cookie deleted");
ok (! cookie_file ($value), "$value deleted from cookie file");

my $badlogin = ['user-name' => 'mariko', password => 'nyanburger'];
$reply = $ua->post ("$url?action=login", $badlogin);
ok ($reply->is_success (), "OK post req with password");
@cookies = $cj->cookies_for ($url);
ok (scalar (@cookies) == 0, "No cookie set with bad password");
$ua->get ("$url?control=stop");
done_testing ();
exit;

sub cookie_file
{
    my ($cookie) = @_;
    my $found;
    my $logins = read_json ("$Bin/logins.json");
    for (@$logins) {
	if ($_->{cookie} eq $cookie) {
	    $found = 1;
	}
    }
    return $found;
}
