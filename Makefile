
TAG ?= $$(git rev-parse HEAD)

build:
	@env GOMODULE111=on find ./cmd/* -maxdepth 1 -type d -exec go build "{}" \;

vet:
	@go vet -v ./...

check: vet

test:
	@go test -v ./...

docker-build:
	@docker build -t harbor.core.skyd.io/skydio-internal/cloud/infrastructure/pagerduty-slack-sync:${TAG} -f build/package/Dockerfile .

docker-publish:
	@docker login harbor.core.skyd.io
	@docker push harbor.core.skyd.io/skydio-internal/cloud/infrastructure/pagerduty-slack-sync:${TAG}

ci: build check test

.PHONY: build vet check test