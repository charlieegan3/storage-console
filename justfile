set shell := ["sh", "-c"]

FILE_PATTERN := "yaml\\|html\\|go\\|sql\\|justfile\\|js\\|css\\|scss"

dev_server:
    GO_ENV=dev go run main.go config.yaml

test:
    go test ./pkg/...

lint:
    golangci-lint run ./...

watch COMMAND:
    find . | grep '{{ FILE_PATTERN }}' | entr -c just {{ COMMAND }}

minio:
    minio server ~/Desktop/minio
