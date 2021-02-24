#!/home/ben/software/install/bin/perl
use warnings;
use strict;
use utf8;
use FindBin '$Bin';
use JSON::Create 'write_json';
my @users = (
    {
	login => 'duncan',
	pass => '12345',
    },
    {
	login => 'tony',
	pass => 'abcde',
    },
    {
	login => 'mariko',
	pass => 'nyan',
    },
);
write_json ("$Bin/users.json", \@users);
