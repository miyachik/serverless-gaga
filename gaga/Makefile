.PHONY: build clean deploy

build:
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/gaga gaga/main.go

clean:
	rm -rf ./bin ./vendor

deploy: clean build
	sls deploy --verbose
	sls s3deploy

