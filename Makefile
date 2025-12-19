GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)

# 取git commit的8位编号，支持通过 make -v VERSION=x.x.x 覆盖
VERSION?=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git describe --tags --always --dirty)

dbIP?=127.0.0.1
dbPort?=3306
dbUser?=pgo
dbPass?=pgo
dbName?=pgo

dbCmd=mysql -h $(dbIP) -P ${dbPort} -u $(dbUser) -p$(dbPass)

# 遍历所有proto文件
# every developer has a Git. run in GitBash.
API_PROTO_FILES=$(shell find ./proto -name *.proto)

# --------------------------------------------------
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

# --------------------------------------------------
.PHONY: env
# 安装依赖
env:
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

	echo "servers:" >> ./openapi.yaml
	echo "    - description: PGO API" >> ./openapi.yaml
	echo "      url: http://127.0.0.1:8080" >> ./openapi.yaml

.PHONY: gorm
gorm:
	$(dbCmd) -e "DROP DATABASE IF EXISTS pgo_build; CREATE DATABASE pgo_build DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"
	for file in ./internal/pkg/db/*.sql; do \
		$(dbCmd) pgo_build < $$file; \
	done

	rm -rf ./internal/pkg/db/model/
	rm -rf ./internal/pkg/db/query/
	gentool \
	-db mysql \
	-dsn "${dbUser}:${dbPass}@tcp(${dbIP}:${dbPort})/pgo_build?charset=utf8mb4&parseTime=True&loc=Local" \
	-outPath internal/pkg/db/query/ \
	-outFile query.go \
	-modelPkgName model \

.PHONY: initDB
initDB:
	$(dbCmd) -e "CREATE DATABASE ${dbName} DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"
	for file in ./internal/pkg/db/*.sql; do \
		$(dbCmd) ${dbName} < $$file; \
	done

.PHONY: reInitDB
# 慎重，这是重置数据库的操作
reInitDB:
	$(dbCmd) -e "DROP DATABASE ${dbName};"
	make initDB

.PHONY: curd
# 根据数据库生成 CURD 代码
curd:
	go run ./tools/genCURD/

.PHONY: build
# build
build: 
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(git rev-parse HEAD) -X main.date=$(date +%Y-%m-%dT%H:%M:%S)" -o ./bin/ ./...

.PHONY: precommit
# 提交生成的代码[*.pb.go, ./client/swagger/*, *.gen.go, *.gen.proto]
precommit:
	git add "openapi.yaml"
	git add "*.pb.go"
	git add "./client/swagger/*"
	git add "*.gen.go"
	git add "*.gen.proto"
	git add "./internal/pkg/db/model/"
	git add "./internal/pkg/db/query/"
	git commit -m "gen: update generated code"
