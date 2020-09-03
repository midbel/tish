# simple commands
echo foobar
echo $FOO
echo dollar \$ pound \# quote \"

# strong quoted string
echo 'foo bar'
echo '"foo bar"'

# weak quoted string
echo "foo bar"
echo "\"foo bar\""
echo foo" <foobar> "bar
echo foo" <$FOO> <$BAR> "bar

# simple sequence of command
echo foo; echo bar
echo foo & echo bar
echo foo | echo bar

# conditional sequence of command
echo foo && echo bar
echo foo || echo bar
