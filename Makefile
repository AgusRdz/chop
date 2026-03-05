.PHONY: build test clean cross

build:
	docker compose run --rm dev go build -o bin/chop .

test:
	docker compose run --rm dev go test ./... -v

clean:
	rm -rf bin/

cross:
	docker compose run --rm dev sh -c "\
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o bin/chop-linux-amd64 . && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='-s -w' -o bin/chop-darwin-amd64 . && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags='-s -w' -o bin/chop-darwin-arm64 . && \
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags='-s -w' -o bin/chop-windows-amd64.exe ."
