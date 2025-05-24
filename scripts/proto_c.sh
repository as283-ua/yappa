#!/bin/bash

rm -r api/gen/*

mkdir -p api/gen/ca api/gen/server api/gen/client
protoc -I=api/proto --go_out=api/gen/ca --go_opt=paths=source_relative api/proto/ca.proto
protoc -I=api/proto --go_out=api/gen/server --go_opt=paths=source_relative api/proto/server.proto
protoc -I=api/proto --go_out=api/gen/client --go_opt=paths=source_relative api/proto/client.proto
