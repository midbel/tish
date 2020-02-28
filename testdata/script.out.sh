testdata
midbel

# read special varaible: UID, $$
# 12345
# 1000

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
# foo-1 foo-2 foo-3 bar-4 bar-5 bar-6

# parameter expansion (part 1)
standard: FOOBAR
length  : 6
suffix  : FOO
prefix  : BAR
replace : F00BAR
substr1 : FOO
substr2 : BAR

# parameter expansion (part 2)
FOOBAR
DEFAULT
DEFAULT
