 #!/usr/bin/env bash

protoc ./proto/service.proto --gocosmos_out=plugins=interface+grpc,Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types:$GOPATH/src --proto_path=./proto -I=$GOPATH/src
protoc ./proto/party.proto --gogofaster_out=$GOPATH/src --proto_path=./proto
protoc ./proto/request.proto --gogofaster_out=$GOPATH/src --proto_path=./proto
protoc ./proto/session.proto --gogofaster_out=$GOPATH/src --proto_path=./proto