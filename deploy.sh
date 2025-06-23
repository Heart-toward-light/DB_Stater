#!/usr/bin/env bash
# @Author : WuWeiJian
# @Date : 2021-09-23 14:09

version="$1"
comment="$2"

# 使用示例以及参数说明
function usage() {
    echo "usage:  sh $0 v1.0.16 '解决mongodb安装失败的bug'
    $1
"
    exit 2
}

[[ "${version}" =~ ^v[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3} ]] || usage "版本: ${version} 无效"
[[ "${comment}AA" == "AA" ]] && usage "必须指定提交说明: \$2"

for tag in $(git tag); do
    [[ "${version}" == "${tag}" ]] && usage "版本标签: ${version} 已经存在"
done

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
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X dbup/cmd/dbupbackup/backupcmd._version=${version}" -o oasis/oasis-linux-amd64/bin/dbupbackup || exit 2
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X dbup/cmd/dbupbackup/backupcmd._version=${version}" -o oasis/oasis-linux-arm64/bin/dbupbackup || exit 2
cd ../..

echo "cd oasis; git submodule add and commit ..."
cd oasis || exit 2
#git checkout master || exit 2
git add . || exit 2
git commit -m "${comment}" || exit 2
git tag "${version}" || exit 2
git push || exit 2
git push origin --tags || exit 2

cd .. || exit 2
git tag "${version}" || exit 2
git push || exit 2
git push origin --tags || exit 2

cd oasis || exit 2
echo "打包amd64..."
sudo chown -R root:root oasis-linux-amd64
tar -zcvf "oasis-linux-amd64-${version}.tar.gz" oasis-linux-amd64 || exit 2
echo "打包arm64..."
sudo chown -R root:root oasis-linux-arm64
tar -zcvf "oasis-linux-arm64-${version}.tar.gz" oasis-linux-arm64 || exit 2

echo "上传amd64..."
curl -ucloudoasis:AP32rzD44cwmaV9UpCjMkW4oQJfT14FbUtso2u -T "oasis-linux-amd64-${version}.tar.gz" "https://af-biz.qianxin-inc.cn/artifactory/qianxin-generic-rc-oasis/oasis-linux-amd64-${version}.tar.gz" || exit 2
echo "上传arm64..."
curl -ucloudoasis:AP32rzD44cwmaV9UpCjMkW4oQJfT14FbUtso2u -T "oasis-linux-arm64-${version}.tar.gz" "https://af-biz.qianxin-inc.cn/artifactory/qianxin-generic-rc-oasis/oasis-linux-arm64-${version}.tar.gz" || exit 2
cd .. || exit 2
echo "完成!"
