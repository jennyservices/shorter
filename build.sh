#!/bin/bash
jenny generate
protoc -I transport/pb transport/pb/shorter.proto --go_out=plugins=grpc:transport/pb
go get ./cmd/...