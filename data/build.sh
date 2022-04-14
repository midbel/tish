echo "run clean cache"
go clean -cache

echo "run get version"
version = $(git tag | tail -n 1)
echo $version
echo "run get date"
build   = $(date -I)
echo $build
buildoptions = "-v" "-trimpath" "-ldflags" "-X main.CmdVersion=${version#v} -X main.CmdBuild=${build}"

bindir = bin
tish   = tish

echo "build ${tish,,} in ${bindir,,}/${tish,,}"
go build -o "${bindir,,}/${tish,,}" "cmd/${tish,,}/main.go"
