#!/bin/sh

rm mapleseed
GOOS=linux GOARCH=amd64 go build
scp mapleseed root@databox1x.com:
# restart it?
