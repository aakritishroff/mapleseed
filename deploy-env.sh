#!/bin/sh

to=$1

if test "$to" = "" ; then
  echo 'What hostname?'
  exit 1
fi

scp admin/debian-setup-go.sh root@${to}:
ssh root@$to "sh debian-setup-go.sh"
sed "s/@@SUBHOST/$1/" < admin/debian-init-script > admin/debian-init-script-$1
scp admin/debian-init-script-$1 root@${to}:/etc/init.d/mapleseed
ssh root@$to "apt-get install rcconf mercurial; rcconf"
ssh root@$to "go get 'code.google.com/p/go.net/websocket'"
