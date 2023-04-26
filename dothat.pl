#!/home/ben/software/install/bin/perl
use Z;
my $verbose = 1;
chdir $Bin or die $!;
my $tag = `git describe --tags`;
my $n = $tag;
$n =~ s!v0\.1\.!!;
$n++;
my $t = "v0.1.$n";
do_system ("git add .;git commit -a -m bababa;git tag $t;git push origin $t",
	   $verbose);
chdir "/home/ben/projects/bagpub" or die $!;
my $file = "go.mod";
my $text = read_text ($file);
$text =~ s!\Q$tag!$t!;
write_text ($file, $text);
do_system ("go get;make", $verbose);
