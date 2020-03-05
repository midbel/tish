write arguments to standard output
usage: echo [-i] [-n] [-h] [arg...]
foobar

# arithmetic errors
# division by zero # echo $((2/0))
# division by zero # echo $((2%0))
# negative shift count: -2 # echo $((1<<-2))
# negative shift count: -2 # echo $((1>>-2))

# filesystem (cd)
..: no such file or directory
foo.txt: not a directory
empty: no such file or directory

testdata: no such file or directory
