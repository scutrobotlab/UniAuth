#!/bin/bash
#try to connect to google to determine whether user need to use proxy
curl www.google.com -o /dev/null --connect-timeout 5 2> /dev/null
if [ $? == 0 ]
then
    echo "Successfully connected to Google, no need to use Go proxy"
else
    echo "Google is blocked, Go proxy is enabled: GOPROXY=https://goproxy.cn,direct"
    export GOPROXY="https://goproxy.cn,direct"
fi

version=$(git describe --tags --abbrev=0)
commit=$(git rev-parse --short HEAD)
commitOffset=$(git rev-list $version..HEAD --count)

ldflags="-w -s -X 'util.version=$version' -X 'util.commit=$commit' -X 'util.commitOffset=$commitOffset'"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags=$ldflags -o server_linux_amd64 .
