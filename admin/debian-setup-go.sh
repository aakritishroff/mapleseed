#!/bin/sh

VERSION=1.3.3

cd /usr/local
if [ ! -d go ]; then
  wget -q https://storage.googleapis.com/golang/go${VERSION}.linux-amd64.tar.gz
  tar xf go${VERSION}.linux-amd64.tar.gz
fi

mkdir -p /root/go/src/github.com/sandhawke/mapleseed
mkdir -p /root/go/src/github.com/sandhawke/pagestore/inmem
mkdir -p /var/log/mapleseed
if [ ! -f /var/log/mapleseed/save.json ]; then
  echo '{ "_members": [] }' > /var/log/mapleseed/save.json
fi

echo 'export GOPATH=/root/go' >> /root/.bashrc
echo 'export GOROOT=/usr/local/go' >> /root/.bashrc
echo 'export PATH=/usr/local/go/bin:$PATH' >> /root/.bashrc
 
. /root/.bashrc

go version