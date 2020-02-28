export HOME="testdata"
echo $HOME
echo -h

NAME=midbel
echo $NAME

echo $$
echo $UID

# define a builtin
alias readfile="echo -i"
readfile < testdata/foo.txt

# try the type builtin
type echo
type testdata
type readfile

unalias readfile

echo foo; echo bar

seq -s ', ' 1 5

echo foobar >&2 # redirect foobar to stderr

export PATH="/bin"
export PATH="$PATH:/sbin"
echo $PATH
