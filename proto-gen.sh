 #!/usr/bin/env bash

protoc ./proto/service.proto --go-grpc_out=$GOPATH/src --go_out=$GOPATH/src --proto_path=./proto -I=$GOPATH/src
protoc ./proto/request.proto --go_out=$GOPATH/src --proto_path=./proto -I=$GOPATH/src
protoc ./proto/controllers.proto --go_out=$GOPATH/src --proto_path=./proto -I=$GOPATH/src
protoc ./proto/session.proto --go_out=$GOPATH/src --proto_path=./proto -I=$GOPATH/src