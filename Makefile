APP := grid-fault-locate-service
IMAGE ?= grid-fault-locate-service:latest
GO ?= go

.PHONY: all build vet fmt test race run clean docker-build docker-run

all: fmt vet test build

build:
	CGO_ENABLED=0 $(GO) build -o bin/$(APP) .

vet:
	$(GO) vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

run:
	$(GO) run .

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm -p 8080:8080 -v $(PWD)/data:/data $(IMAGE)

clean:
	rm -rf bin data
