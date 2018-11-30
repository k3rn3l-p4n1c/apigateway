help:
	@echo "Please use \`make <ROOT>' where <ROOT> is one of"
	@echo "  test                 to run test cases"
	@echo "  clean                to remove generated files"
	@echo "  dependencies         to install dependencies"
	@echo "  update               to update dependencies"
	@echo "  build                to build binary"

clean:
	rm -rf apigateway

update-dependencies:
	dep version 2> /dev/null || curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	git config --global url.${GITLAB_SUPERSEDED_URL}.insteadOf "https://git.cafebazaar.ir/"
	dep ensure -update

dependencies:
	dep version 2> /dev/null || curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	git config --global url.${GITLAB_SUPERSEDED_URL}.insteadOf "https://git.cafebazaar.ir/"
	dep ensure

build: *.go */*.go */*/*.go Gopkg.lock
	$(GO_VARS) $(GO) run writeversion.go
	$(GO_VARS) $(GO) build -i -o="apigateway" -ldflags="$(LD_FLAGS)" $(ROOT)/cmd

test: *.go */*.go */*/*.go
	$(GO_VARS) $(GO) test $(GO_PACKAGES) -cover -timeout 5s -v -failfast && echo -e "\nTesting is passed."

## Vars ##########################################################
ROOT := github.com/k3rn3l-p4n1c/apigateway
GO_VARS = ENABLE_CGO=0 GOARCH=amd64
GO_PACKAGES := $(shell go list ./... | grep -v vendor)
GO ?= go
GIT ?= git
COMMIT := $(shell $(GIT) rev-parse HEAD)
VERSION ?= $(shell $(GIT) describe --tags ${COMMIT} 2> /dev/null || echo "$(COMMIT)")
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
LD_FLAGS := -X $(ROOT).Version=$(VERSION) -X $(ROOT).Commit=$(COMMIT) -X $(ROOT).BuildTime=$(BUILD_TIME) -X $(ROOT).Title=apigateway
CONFIG_FILE := `pwd`/local.sample.yaml
