GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)

# 取git commit的8位编号，支持通过 make -v VERSION=x.x.x 覆盖
VERSION?=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git describe --tags --always --dirty)

dbIP?=127.0.0.1
dbUser?=pgo
dbPass?=pgo
dbName?=pgo

dbCli?=go run ./cmd/pgo/ psql
# dbCli?=psql

dbCmd=export PGPASSWORD=${dbPass}; \
	${dbCli} --host $(dbIP) -U ${dbUser}

# 遍历所有proto文件
# every developer has a Git. run in GitBash.
API_PROTO_FILES=$(shell find ./proto -name *.proto)

.PHONY: init
# 安装依赖
init:
# wget https://github.com/protocolbuffers/protobuf/releases/download/v28.1/protoc-28.1-linux-x86_64.zip
# unzip protoc-28.1-linux-x86_64.zip -d /usr/local
# go env -w GOPROXY=https://goproxy.cn,direct
# git config core.autocrlf false
# git config core.eol lf
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	go install gorm.io/gen/tools/gentool@latest
	go mod tidy

.PHONY: api
# generate api proto
api:
	rm -f ./api/*.pb.go
	protoc --proto_path=./proto/ \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./api/ \
		--go-http_out=paths=source_relative:./api/ \
		--go-grpc_out=paths=source_relative:./api/ \
		--go-errors_out=paths=source_relative:./api/ \
		--openapi_out=fq_schema_naming=true,default_response=false:. \
		$(API_PROTO_FILES) \

	echo servers: >> ./openapi.yaml
	echo     - description: IN Gen2 Open API >> ./openapi.yaml
	echo       url: http://127.0.0.1:8080 >> ./openapi.yaml

.PHONY: gorm
# 通过 gorm-gen 生成数据库访问代码
gorm:
	rm -rf ./internal/pkg/db/model/
	rm -rf ./internal/pkg/db/query/
	${dbCmd} -d postgres -c "DROP DATABASE IF EXISTS pgo_build;"
	${dbCmd} -d postgres -c "CREATE DATABASE pgo_build;"
	for file in ./internal/pkg/db/*.sql; do \
		${dbCmd} -d pgo_build -f $$file; \
	done

	gentool \
	-db postgres \
	-dsn "host=$(dbIP) user=${dbUser} password=${dbPass} dbname=pgo_build port=5432 sslmode=disable" \
	-outPath ./internal/pkg/db/query \
	-modelPkgName "model"

.PHONY: initDB
initDB:
	${dbCmd} -d postgres -c "CREATE DATABASE ${dbName};"
	for file in internal/pkg/db/*.sql; do \
		${dbCmd} -d ${dbName} -f $$file; \
	done

.PHONY: reInitDB
# 慎重，这是重置数据库的操作
reInitDB:
	${dbCmd} -d postgres -c "DROP DATABASE ${dbName};"
	make initDB

.PHONY: curd
# 根据数据库生成 CURD 代码
curd:
	go run ./tools/genCURD/

.PHONY: build
# build
build: 
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(git rev-parse HEAD) -X main.date=$(date +%Y-%m-%dT%H:%M:%S)" -o ./bin/ ./...

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
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-10s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

# 设置help为默认目标，平时默认目标是第一个目标
.DEFAULT_GOAL := help

.PHONY: precommit
# 提交生成的代码[*.pb.go, ./client/swagger/*, *.gen.go, *.gen.proto]
precommit:
	git add "*.pb.go"
	git add "./client/swagger/*"
	git add "*.gen.go"
	git add "*.gen.proto"
	git add "./internal/pkg/db/model/"
	git add "./internal/pkg/db/query/"
	git commit -m "gen: update generated code"
