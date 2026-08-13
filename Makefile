.PHONY: build test race vet fmt fmt-check run simulate results evidence docker-build

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

fmt-check:
	test -z "$$(gofmt -l $$(find cmd internal -name '*.go' -type f))"

run:
	go run ./cmd/server

simulate:
	go run ./cmd/simulator -scenario scenarios/balanced.json

results:
	./scripts/generate-experiment-results.sh

evidence: fmt-check vet test race build results
	EVIDENCE_CHECKS_PASSED=1 ./scripts/generate-evidence-manifest.sh

docker-build:
	docker build -t cluster-process-manager .
