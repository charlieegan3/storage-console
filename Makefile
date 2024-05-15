FILE_PATTERN := 'yaml\|html\|go\|sql\|Makefile\|js\|css\|scss'
dev_server:
	find . | grep $(FILE_PATTERN) | GO_ENV=dev entr -c -r go run main.go config.yaml

watch_test:
	find . | grep $(FILE_PATTERN) | entr -c go test ./pkg/...

watch_lint:
	find . | grep $(FILE_PATTERN) | entr -c golangci-lint run ./...

watch_server:
	find . | grep $(FILE_PATTERN) | entr -c -r go run main.go config.yaml

minio:
	minio server ~/Desktop/minio
