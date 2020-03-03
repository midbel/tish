export HOME="/testdata"
echo $HOME
echo -h;

NAME=midbel
echo $NAME

echo $$;
echo $UID

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
echo $((-2+2))
echo $((1 << 2))
echo $((8 >> 2))

:'the quick brown fox
jumps over
the lazy dog
'

echo foo && echo bar
echo foo || echo bar
false || echo bar

# braces expansion
echo {foo,bar}
echo {foo-{1,2,3}, bar-{4,5,6}}

# parameter expansion (part 1)
FOOBAR=FOOBAR
echo "standard: ${FOOBAR}"
echo "length  : ${#FOOBAR}"
echo "suffix  : ${FOOBAR%BAR}"
echo "prefix  : ${FOOBAR#FOO}"
echo "replace : ${FOOBAR//O/0}"
echo "substr1 : ${FOOBAR:0:3}"
echo "substr2 : ${FOOBAR:(-3):0}"

THREE=3
ZERO=0
echo "substr3 : ${FOOBAR:$ZERO:$THREE}"

# parameter expansion (part 2)
echo "${FOOBAR:-DEFAULT}"
echo "${DEFAULT:=DEFAULT}"
echo "${FOOBAR:+DEFAULT}"

# filesystem
cd /
echo $PWD
pwd
cd .. # try to go somewhere outside of the filesystem
cd testdata
echo $PWD
pwd
cd ..
echo $PWD
pwd
cd
echo $PWD
pwd
cd foo.txt # can not cd into a regular file
cd empty # can not cd into a directory that does not exist
