export HOME="testdata"
echo $HOME
echo -h

NAME=midbel
echo $NAME

# echo $$
# echo $UID

# define a builtin
alias readfile="echo -i"
readfile < testdata/foo.txt

# try the type builtin
type echo
type testdata
type readfile

unalias readfile
unalias -a

echo foo; echo bar

seq -s ', ' 1 5

echo foobar >&2 # redirect foobar to stderr

export PATH="/bin"
echo $PATH
export PATH="$PATH:/sbin"
echo $PATH

local FOO=FOO BAR=BAR
echo $FOO $BAR

# arithmetic
echo $((2*2))
echo $((2+2))
echo $((2/2))
echo $((2%2))
# echo $((-2+2))
echo $((1 << 2))
echo $((8 >> 2))

:'the quick brown fox
jumps over
the lazy dog
'

echo foo && echo bar
echo foo || echo bar
false || echo bar
