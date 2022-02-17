.PHONY: tools build clean

BINARY="cupid"

all: tools build

tools:
	@go fmt ./...
	@go vet ./...

build:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o output/${BINARY} main.go
	@chmod +x output/${BINARY}

run-sniff:
	@go run main.go sniff -c configs/configs.yaml

debug-sniff:
	@go run main.go sniff -c configs/debug.yaml

run-seckill:
	@go run main.go seckill -c configs/configs.yaml

debug-seckill:
	@go run main.go seckill -c configs/debug.yaml

clean:
	@rm -rf tmp
	@rm -rf logs
	@rm -rf output

help:
	@echo "make - 格式化代码和静态检查代码，然后编译生成二进制文件"
	@echo "make tools - 格式化代码和静态检查代码"
	@echo "make build - 编译生成二进制文件"
	@echo "make clean - 移除二进制文件和日志文件"
