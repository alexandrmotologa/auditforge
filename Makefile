.PHONY: all build build-web test proto clean run

all: build

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/api/grpc/proto/audit.proto

build-web:
	cd web && npm install && npm run build

build:
	go build -o bin/auditforge.exe ./cmd/auditforge
	go build -o bin/auditverify.exe ./cmd/auditverify

test:
	go test -v ./...

clean:
	rm -rf bin/ internal/ui/dist/*

run: build
	./bin/auditforge.exe --config config.yaml
