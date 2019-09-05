# TODO: Fix this on windows.
ALL_SRC := $(shell find . -name '*.go' \
								-not -path './vendor/*' \
								-not -path '*/gen-go/*' \
								-type f | sort)
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GOTEST_OPT?=-v -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
GOTEST=go test
GOFMT=gofmt
GOLINT=golint
GOVET=go vet
EMBEDMD=embedmd
ADDLICENCESE= addlicense
STATICCHECK=staticcheck
# TODO decide if we need to change these names.
README_FILES := $(shell find . -name '*README.md' | sort | tr '\n' ' ')

.DEFAULT_GOAL := fmt-lint-vet-embedmd-test

.PHONY: fmt-lint-vet-embedmd-test
fmt-lint-vet-embedmd-test: addlicense fmt lint vet embedmd staticcheck test

.PHONY: travis-ci
travis-ci: fmt lint vet embedmd test test-386 test-with-cover

.PHONY: test-with-cover
test-with-cover:
	@echo Verifying that all packages have test files to count in coverage
	@scripts/check-test-files.sh $(subst contrib.go.opencensus.io/exporter/stackdriver,./,$(ALL_PKGS))
	@echo pre-compiling tests
	@time go test -i $(ALL_PKGS)
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

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
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)

.PHONY: fmt
fmt:
	@FMTOUT=`$(GOFMT) -s -l $(ALL_SRC) 2>&1`; \
	if [ "$$FMTOUT" ]; then \
		echo "$(GOFMT) FAILED => gofmt the following files:\n"; \
		echo "$$FMTOUT\n"; \
		exit 1; \
	else \
	    echo "Fmt finished successfully"; \
	fi

.PHONY: lint
lint:
	@LINTOUT=`$(GOLINT) $(ALL_PKGS) 2>&1`; \
	if [ "$$LINTOUT" ]; then \
		echo "$(GOLINT) FAILED => clean the following lint errors:\n"; \
		echo "$$LINTOUT\n"; \
		exit 1; \
	else \
	    echo "Lint finished successfully"; \
	fi

.PHONY: vet
vet:
    # TODO: Understand why go vet downloads "github.com/google/go-cmp v0.2.0"
	@VETOUT=`$(GOVET) ./... | grep -v "go: downloading" 2>&1`; \
	if [ "$$VETOUT" ]; then \
		echo "$(GOVET) FAILED => go vet the following files:\n"; \
		echo "$$VETOUT\n"; \
		exit 1; \
	else \
	    echo "Vet finished successfully"; \
	fi
	
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

.PHONY: addlicense
addlicense:
	@ADDLICENCESEOUT=`$(ADDLICENCESE) -y 2019 -c 'OpenTelemetry Authors' $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENCESEOUT" ]; then \
			echo "$(ADDLICENCESE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENCESEOUT\n"; \
			exit 1; \
		else \
			echo "Add License finished successfully"; \
		fi

.PHONY: staticcheck
staticcheck:
	$(STATICCHECK) ./...

.PHONY: install-tools
install-tools:
	GO111MODULE=on go install \
		golang.org/x/lint/golint \
		golang.org/x/tools/cmd/goimports \
		github.com/google/addlicense \
		github.com/rakyll/embedmd \
		honnef.co/go/tools/cmd/staticcheck
