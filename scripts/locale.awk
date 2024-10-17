# This simple script converts the output of `locale -a -v` into a 3-tuple
# file that has "abbrev shortname encoding".
# It can be used with `locale -a -v | awk -f locale.awk` on a Debian system where
# `sudo apt install locales-all`

BEGIN { RS="\n\n"; FS="\n"; }
/locale:/ {
    #print "$1=", $1, "$11=", $11,"$12=", $12, "\n";
    split($1, locale_info, " ");  # Split the first line to get the locale name
    #print "locale_info $1=", locale_info[1], " $2=", locale_info[2],"\n";
    locale = locale_info[2];       # Extract the locale name (second item in the split array)
    for (i=3; i<=NF; i++) {
        if ($i ~ /codeset/) {
            codeset_line=$i
            break
        }
    }
    split(codeset_line, codeset_info, "|");
    split(locale, locale_pieces, ".");
    #print "locale_pieces[1]=",locale_pieces[1];
    print locale, locale_pieces[1], codeset_info[2];
}
