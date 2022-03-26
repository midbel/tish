total = $(find -type f -name "*.go" -exec cat {} \; | egrep -v '^$' | wc -l)
echo "${total:>:5} LoC"
