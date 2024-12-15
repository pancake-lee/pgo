GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
# 取git commit的8位编号
VERSION=$(shell git describe --tags --always)
dbIP=go-pg
# dbIP=192.168.3.111 

# 遍历所有proto文件
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
# wget https://github.com/protocolbuffers/protobuf/releases/download/v28.1/protoc-28.1-linux-x86_64.zip
# unzip protoc-28.1-linux-x86_64.zip -d /usr/local
	go env -w GOPROXY=https://goproxy.cn,direct
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	go install gorm.io/gen/tools/gentool@latest


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
	rm -rf ./pkg/db/dao/
	export PGPASSWORD="gogogo"; \
	psql -h $(dbIP) -U gogogo -d postgres -c "DROP DATABASE gogogo_build;"
	export PGPASSWORD="gogogo"; \
	psql -h $(dbIP) -U gogogo -d postgres -c "CREATE DATABASE gogogo_build;"
	for file in pkg/db/*.sql; do \
		export PGPASSWORD="gogogo"; \
        psql -h $(dbIP) -U gogogo -d gogogo_build -f $$file; \
    done

	gentool \
	-db postgres \
	-dsn "host=$(dbIP) user=gogogo password=gogogo dbname=gogogo_build port=5432 sslmode=disable" \
	-outPath ./pkg/db/dao/query \
	-modelPkgName "pkg/db/dao/model"

# 慎重，这是重置数据库的操作，交互输入密码
initDB:
# psql -h $(dbIP) -U gogogo -d postgres -c "CREATE DATABASE gogogo;"
	for file in pkg/db/*.sql; do \
		export PGPASSWORD="gogogo"; \
		psql -h $(dbIP) -U gogogo -d gogogo -f $$file; \
	done

curd:
	go run ./tools/genCURD/

.PHONY: build
# build
build: 
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: all
# generate all
all: gorm curd build

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
