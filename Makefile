GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
# 取git commit的8位编号
VERSION=$(shell git describe --tags --always)

dbIP=127.0.0.1
# dbIP=go-pg
# dbIP=192.168.3.111 

# 遍历所有proto文件
# 这段在window上找git的代码在某些机器上有问题，我们直接用git-bash运行就好了
# ifeq ($(GOHOSTOS), windows)
# 	#the `find.exe` is different from `find` in bash/shell.
# 	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
# 	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
# 	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
# 	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
# 	API_PROTO_FILES=$(shell $(Git_Bash) -c "find ./api/proto -name *.proto")
# else
API_PROTO_FILES=$(shell find ./proto -name *.proto)
# endif

.PHONY: default
default:all

.PHONY: init
# init env
init:
# wget https://github.com/protocolbuffers/protobuf/releases/download/v28.1/protoc-28.1-linux-x86_64.zip
# unzip protoc-28.1-linux-x86_64.zip -d /usr/local
# go env -w GOPROXY=https://goproxy.cn,direct
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	go install gorm.io/gen/tools/gentool@latest
	go mod tidy
	git config core.autocrlf false
	git config core.eol lf

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
	export PGPASSWORD="pgo"; \
	psql -h $(dbIP) -U pgo -d postgres -c "DROP DATABASE IF EXISTS pgo_build;"
	export PGPASSWORD="pgo"; \
	psql -h $(dbIP) -U pgo -d postgres -c "CREATE DATABASE pgo_build;"
	for file in ./internal/pkg/db/*.sql; do \
		export PGPASSWORD="pgo"; \
		psql -h $(dbIP) -U pgo -d pgo_build -f $$file; \
	done

	gentool \
	-db postgres \
	-dsn "host=$(dbIP) user=pgo password=pgo dbname=pgo_build port=5432 sslmode=disable" \
	-outPath ./internal/pkg/db/query \
	-modelPkgName "model"

# 慎重，这是重置数据库的操作，交互输入密码
initDB:
# psql -h $(dbIP) -U pgo -d postgres -c "CREATE DATABASE pgo;"
	for file in pkg/db/*.sql; do \
		export PGPASSWORD="pgo"; \
		psql -h $(dbIP) -U pgo -d pgo -f $$file; \
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
