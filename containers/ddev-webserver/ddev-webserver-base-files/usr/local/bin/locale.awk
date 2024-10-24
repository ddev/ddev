BEGIN { RS="\n\n"; FS="\n"; }
/locale:/ {
    #print "$1 is ", $1, "\n";
    split($1, locale_info, " ");  # Split the first line to get the locale name
    #print "locale_info $1=", locale_info[1], " $2=", locale_info[2],"\n"; 
    locale = locale_info[2];       # Extract the locale name (second item in the split array)
}
$12 ~ /codeset/ {
    split($12, codeset_info, "|");  # Split the codeset line using the pipe delimiter
    codeset = codeset_info[2];     # Extract the codeset value (second item in the split array)
    gsub(/^ +| +$/, "", codeset);  # Remove any leading or trailing whitespace
    print "Locale:", locale, "Codeset:", codeset;
}
