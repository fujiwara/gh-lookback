.PHONY: clean test

gh-lookback: go.* *.go
	go build -o $@ ./cmd/gh-lookback

clean:
	rm -rf gh-lookback dist/

test:
	go test -v ./...

install:
	go install github.com/fujiwara/gh-lookback/cmd/gh-lookback

dist:
	goreleaser build --snapshot --clean
