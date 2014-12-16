#!/bin/sh

host=$1    # we need to know our nominal hostname for the pod server
# or maybe this should be a config file?

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

mkdir -p /root/go/src/github.com/sandhawke/mapleseed
mkdir -p /root/go/src/github.com/sandhawke/pagestore/inmem
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

apt-get update
apt-get --yes install mercurial supervisor screen htop
go get 'code.google.com/p/go.net/websocket'

cd
sed "s/@@HOST/$host/g" < supervisord.conf.template > /etc/supervisor/conf.d/mapleseed.conf
supervisorctl reread
supervisorctl update
