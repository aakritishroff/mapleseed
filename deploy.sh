#!/bin/sh

to=$1

if test "$to" = "" ; then
  echo 'What hostname?'
  exit 1
fi

scp *.go root@${to}:/root/go/src/github.com/sandhawke/mapleseed
scp ../pagestore/inmem/*.go root@${to}:/root/go/src/github.com/sandhawke/pagestore/inmem
ssh root@${to} "cd go/src/github.com/sandhawke/mapleseed && go build && go test && go install && /etc/init.d/mapleseed restart"
