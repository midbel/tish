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
# /bin:/sbin
