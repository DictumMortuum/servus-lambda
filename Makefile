package: build
	zip servus-lambda.zip main

build: format
	CGO_ENABLED=0 go build -trimpath -buildmode=pie -mod=readonly -modcacherw -ldflags="-s -w" -o main

format:
	gofmt -s -w .
