#!/bin/sh
# generate .pb.go file from .proto file.
set -e
protoc --go_out=. --go-grpc_out=. test.proto
echo '
//go:generate ./generate.sh
' >> test.pb.go
goimports -w test.pb.go

