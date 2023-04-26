#!/home/ben/software/install/bin/perl
use Z;
my $verbose = 1;
chdir $Bin or die $!;
my $tag = `git describe --tags`;
my $n = $tag;
$n =~ s!v0\.1\.!!;
$n++;
my $tag = "v0.1.$n";
do_system ("git add .;git commit -a -m bababa;git tag $tag;git push origin $tag", $verbose);
