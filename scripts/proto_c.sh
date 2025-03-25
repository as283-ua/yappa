#!/bin/bash

rm api/gen/*
protoc -I=api/proto --go_out=api/gen --go_opt=paths=source_relative api/proto/*.proto
