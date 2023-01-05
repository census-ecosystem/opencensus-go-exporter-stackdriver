# TODO: Fix this on windows.
ALL_SRC := $(shell find . -name '*.go' \
								-not -path '*/internal/testpb/*' \
								-not -name 'tools.go' \
								-type f | sort)
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GOTEST_OPT?=-v -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
GOTEST=go test
EMBEDMD=embedmd
STATICCHECK=staticcheck
# TODO decide if we need to change these names.
README_FILES := $(shell find . -name '*README.md' | sort | tr '\n' ' ')

.DEFAULT_GOAL := default-goal

.PHONY: default-goal
default-goal: lint embedmd staticcheck test

# TODO: enable test-with-cover when find out why "scripts/check-test-files.sh: 4: set: Illegal option -o pipefail"
.PHONY: ci
ci: embedmd test test-386 test-with-coverage

all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: test-386
test-386:
	GOARCH=386 $(GOTEST) -v -timeout 30s $(ALL_PKGS)

.PHONY: test-with-coverage
test-with-coverage:
	@echo pre-compiling tests
	@go test -i $(ALL_PKGS)
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: test-with-cover
test-with-cover:
	@echo Verifying that all packages have test files to count in coverage
	@scripts/check-test-files.sh $(subst contrib.go.opencensus.io/exporter/stackdriver,./,$(ALL_PKGS))
	@echo pre-compiling tests
	@go test -i $(ALL_PKGS)
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: lint
lint:
	golangci-lint run

.PHONY: embedmd
embedmd:
	@EMBEDMDOUT=`$(EMBEDMD) -d $(README_FILES) 2>&1`; \
	if [ "$$EMBEDMDOUT" ]; then \
		echo "$(EMBEDMD) FAILED => embedmd the following files:\n"; \
		echo "$$EMBEDMDOUT\n"; \
		exit 1; \
	else \
	    echo "Embedmd finished successfully"; \
	fi

.PHONY: staticcheck
staticcheck:
	$(STATICCHECK) ./...

.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
	go install github.com/rakyll/embedmd@latest
	go install honnef.co/go/tools/cmd/staticcheck@2022.1.3
