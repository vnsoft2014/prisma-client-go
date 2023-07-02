#!/bin/sh

set -eux

CI=${CI:-false}
S3_BUCKET="prisma-photongo"
pkg_version=5.8.1

v="$1"

# do nothing if the version already exists
processed_name="prisma-cli-$v-processed.txt"
aws s3 ls "s3://$S3_BUCKET/$processed_name" && echo "Version $v already exists. Skipping." && exit 0

echo "Building Prisma CLI $v"

mkdir -p build
cd build
yarn init --yes
yarn add "pkg@$pkg_version" "prisma@$v" "@prisma/client@$v"
yarn prisma version

mkdir -p node_modules/prisma/node_modules/@prisma/engines
cp -R node_modules/@prisma/engines/* node_modules/prisma/node_modules/@prisma/engines

npx pkg -t node16-linuxstatic-x64,node16-darwin-x64,node16-win-x64,node16-linuxstatic-arm64,node16-darwin-arm64,node16-win-arm64 node_modules/prisma

version=$(npx prisma version | grep '^\(prisma \)' | cut -d : -f 2 | cut -d " " -f 2)

# abort if the installed version does not equal the release version
if [ "$version" != "$v" ]; then
  echo "Version mismatch: $version != $v"
  exit 1
fi

# test
if [[ $CI == 'true' ]]; then
  echo 'Testing binary'
  ./prisma-linuxstatic-x64 --version
else
  echo 'Skipping tests'
fi

mkdir -p out/

mv prisma-macos-x64 "out/prisma-cli-$version-darwin-x64"
mv prisma-linuxstatic-x64 "out/prisma-cli-$version-linux-x64"
mv prisma-win-x64.exe "out/prisma-cli-$version-windows-x64.exe"
mv prisma-macos-arm64 "out/prisma-cli-$version-darwin-arm64"
mv prisma-linuxstatic-arm64 "out/prisma-cli-$version-linux-arm64"
mv prisma-win-arm64.exe "out/prisma-cli-$version-windows-arm64.exe"

cd out/

gzip -f "prisma-cli-$version-darwin-x64"
gzip -f "prisma-cli-$version-linux-x64"
gzip -f "prisma-cli-$version-windows-x64.exe"
gzip -f "prisma-cli-$version-darwin-arm64"
gzip -f "prisma-cli-$version-linux-arm64"
gzip -f "prisma-cli-$version-windows-arm64.exe"

echo "Uploading Prisma CLI $version"

aws s3 cp . "s3://$S3_BUCKET" --recursive --acl public-read
# make sure all files were successfully uploaded before marking the version as processed
touch "$processed_name"
aws s3 cp "$processed_name" "s3://$S3_BUCKET" --acl public-read

echo "Successfully published Prisma CLI $version"
