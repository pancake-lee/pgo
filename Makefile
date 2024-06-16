GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find pkg/proto -name *.proto")
else
	API_PROTO_FILES=$(shell find pkg/proto -name *.proto)
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest

.PHONY: api
# generate api proto
api:
	protoc --proto_path=./pkg/proto/ \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./pkg/proto/api/ \
 	       --go-http_out=paths=source_relative:./pkg/proto/api/ \
 	       --go-grpc_out=paths=source_relative:./pkg/proto/api/ \
           --go-errors_out=paths=source_relative:./pkg/proto/api/ \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: gorm
# 通过 gorm-gen 生成数据库访问代码
gorm:
	gentool \
	-db postgres \
	-c ./configs/gentool.yaml \
	-outPath ./pkg/db/dao/query \
	-modelPkgName "pkg/db/dao/model"

.PHONY: build
# build
build: 
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: all
# generate all
all: gorm api build

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
