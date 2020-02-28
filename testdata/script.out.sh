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
4
4
1
0
# 0
4 # $((1<<2))
2 # $((8<<2))

# logical and
foo
bar
# logical or
foo
bar
