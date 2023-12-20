echo stdout > foo.sh
echo stdout >> foo.sh
echo stderr 2> foo.sh
echo stderr 2>> foo.sh
echo both &> foo.sh
echo both &>> foo.sh

echo out2err >&2
echo err2out 2>&1

ls foobar > dir  2>&1
ls foobar 2>&1 > dir