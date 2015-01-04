#!/bin/sh

host=$1

if test "$host" = "" ; then
  echo 'What hostname?'
  exit 1
fi

# which version of golang do we want?
VERSION=1.3.3

cd /usr/local
if [ ! -d go ]; then
  wget -q https://storage.googleapis.com/golang/go${VERSION}.linux-amd64.tar.gz
  tar xf go${VERSION}.linux-amd64.tar.gz
fi

mkdir -p /root/go/src/github.com/aakritishroff/mapleseed
mkdir -p /root/go/src/github.com/aakritishroff/data/inmem
mkdir -p /var/log/mapleseed
if [ ! -f /var/log/mapleseed/save.json ]; then
  echo '{ "_members": [] }' > /var/log/mapleseed/save.json
fi

# sorry, but running this many times just adds these.  no real harm.
echo 'export GOPATH=/root/go' >> /root/.bashrc
echo 'export GOROOT=/usr/local/go' >> /root/.bashrc
echo 'export PATH=/usr/local/go/bin:$PATH' >> /root/.bashrc
 
. /root/.bashrc

go version

sed "s/@@SUBHOST/$host/" < debian-init-script > /etc/init.d/mapleseed
apt-get install rcconf mercurial
rcconf --on mapleseed
go get 'code.google.com/p/go.net/websocket'
