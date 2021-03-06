/testdata
midbel

# read special varaible: UID, $$
12345
1000

# read foo.txt with alias
foo

# test builtin type
echo: builtin
testdata: directory
readfile: alias

# test echo x; echo y
foo
bar

# test builtin seq
1, 2, 3, 4, 5

# export PATH
/bin
# export PATH
/bin:/sbin

# echo variable declared with local
FOO BAR

# arithmetic
4 # $((2*2))
4 # $((2+2))
1 # $((2/2))
0 # $((2%2))
0 # $((-2+2))
4 # $((1<<2))
2 # $((8<<2))

# logical and
foo
bar
# logical or
foo
bar

# braces expansion
foo bar
foo-1 foo-2 foo-3 bar-4 bar-5 bar-6

# parameter expansion (part 1)
standard: FOOBAR
length  : 6
suffix  : FOO
prefix  : BAR
replace : F00BAR
substr1 : FOO
substr2 : BAR
substr3 : FOO

# parameter expansion (part 2)
FOOBAR
DEFAULT
DEFAULT

# filesystem
/ # cd / -> echo $WD
/ # cd / -> pwd
/testdata # cd testdata -> echo $PWD
/testdata # cd testdata -> pwd
/ # cd .. -> echo $PWD
/ # cd .. -> pwd
/testdata # cd -> echo $PWD
/testdata # cd -> pwd

# command substitutition
foobar foo bar barfoo # echo foobar $(echo foo $(echo bar)) barfoo
foobar # $(echo echo foobar)

# subshell
/testdata # (SUB=SUBSHELL; cd testdata; pwd; echo $SUB)
SUBSHELL # (SUB=SUBSHELL; cd testdata; pwd; echo $SUB)
/ # pwd
NOT AVAILABLE # echo ${SUB:-NOT AVAILABLE}

# builtin(s)
foobar # builtin echo foobar
source # echo $SOURCE

# chroot
/
/testdata
