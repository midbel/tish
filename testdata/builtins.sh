export FOO=BAR
echo $FOO

command echo.exe "from command foobar"

enable -n echo
enable -p
enable -p -a
builtin echo "from builtin foobar"