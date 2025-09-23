IMAGE ?= k8s-watcher:latest
NAMESPACE ?= default

.PHONY: build
build:
	go build -o bin/watcher ./cmd/watcher

.PHONY: container-build
container-build:
	podman build -t $(IMAGE) .

.PHONY: container-push
container-push:
	podman push $(IMAGE)

.PHONY: deploy
deploy:
	kubectl apply -f deploy/k8s-watcher.yaml

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/k8s-watcher.yaml

.PHONY: logs
logs:
	kubectl logs -f deployment/k8s-watcher -n $(NAMESPACE)

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: all
all: fmt vet test build