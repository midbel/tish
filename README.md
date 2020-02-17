# tish

tish is a tiny shell interpreter largely inspired by bash. But tish is not bash and does not try to be compliant with the bash
(even if most of its syntax is inspired by it) and does not try to be compliant with the shell specification of the opengroup.

Today, tish is only made of a scanner and a parser. Rest of the code will be written to be able to execute shell script.

The element syntax already supported/recognized by tish are:

* simple command: echo foobar
* pipeline: echo foobar | cat
* conditional commands: echo foo && echo bar, echo foo || echo bar
* command substitution: echo foobar $(echo foobar $(echo foobar)))
* arithmetic expression: echo $((1+1))
* braces expansion: pre-{foo,bar-{hello,world}}-post
* parameter expansion: trim prefix, suffix, replacement, slicing, length,...
* 9 kind of redirections: <, >, >>, 2>>, 2>&1,...
* maybe other constructs will be added later

