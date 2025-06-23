#!/usr/bin/env bash
# @Author : WuWeiJian
# @Date : 2021-09-23 19:29

version="$1"
[[ "${version}AA" == "AA" ]] && version="test-$(date '+%F-%H-%M-%S')"

echo "git pull ..."
git pull || exit 2
echo "git submodule update ..."
git submodule update --remote || exit 2
echo "cd oasis; git pull"
cd oasis || exit 2
git checkout master || exit 2
git pull || exit 2

cd .. || exit 2

echo "go build ..."
go mod tidy || exit 2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X dbup/cmd._version=${version}" -o oasis/oasis-linux-amd64/bin/dbup || exit 2
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X dbup/cmd._version=${version}" -o oasis/oasis-linux-arm64/bin/dbup || exit 2
cd cmd/dbupbackup/ || exit 2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X dbup/cmd/dbupbackup/backupcmd._version=${version}" -o ../../oasis/oasis-linux-amd64/bin/dbupbackup || exit 2
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X dbup/cmd/dbupbackup/backupcmd._version=${version}" -o ../../oasis/oasis-linux-arm64/bin/dbupbackup || exit 2
cd ../..
echo "完成!"
