.PHONY: test drone-ecs-deploy build fmt vet errcheck lint install update release-dirs release-build release-copy release-check release coverage

DIST := dist
EXECUTABLE := drone-ecs-deploy
GO ?= go

# for dockerhub
DEPLOY_ACCOUNT := moneysmartco
DEPLOY_IMAGE := $(EXECUTABLE)
GOFMT ?= gofmt "-s"

TARGETS ?= linux darwin windows
PACKAGES ?= $(shell $(GO) list ./... | grep -v /vendor/)
GOFILES := $(shell find . -name "*.go" -type f -not -path "./vendor/*")
SOURCES ?= $(shell find . -name "*.go" -type f)
TAGS ?=
LDFLAGS ?= -X 'main.Version=$(VERSION)' -X 'main.build=$(NUMBER)'
TMPDIR := $(shell mktemp -d 2>/dev/null || mktemp -d -t 'tempdir')

ifneq ($(shell uname), Darwin)
	EXTLDFLAGS = -extldflags "-static" $(null)
else
	EXTLDFLAGS =
endif

ifneq ($(DRONE_TAG),)
	VERSION ?= $(DRONE_TAG)
endif

ifneq ($(DRONE_BUILD_NUMBER),)
	NUMBER ?= $(DRONE_BUILD_NUMBER)
else
	VERSION ?= $(shell git describe --tags --always || git rev-parse --short HEAD)
endif

all: build

.PHONY: fmt-check
fmt-check:
	@diff=$$($(GOFMT) -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;

.PHONY: test-vendor
test-vendor:
	@hash govendor > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/kardianos/govendor; \
	fi
	govendor list +unused | tee "$(TMPDIR)/wc-gitea-unused"
	[ $$(cat "$(TMPDIR)/wc-gitea-unused" | wc -l) -eq 0 ] || echo "Warning: /!\\ Some vendor are not used /!\\"

	govendor list +outside | tee "$(TMPDIR)/wc-gitea-outside"
	[ $$(cat "$(TMPDIR)/wc-gitea-outside" | wc -l) -eq 0 ] || exit 1

	govendor status || exit 1

fmt:
	$(GOFMT) -w $(GOFILES)

vet:
	$(GO) vet $(PACKAGES)

errcheck:
	@hash errcheck > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/kisielk/errcheck; \
	fi
	errcheck $(PACKAGES)

lint:
	@hash golint > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/golang/lint/golint; \
	fi
	for PKG in $(PACKAGES); do golint -set_exit_status $$PKG || exit 1; done;

unconvert:
	@hash unconvert > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mdempsky/unconvert; \
	fi
	for PKG in $(PACKAGES); do unconvert -v $$PKG || exit 1; done;

test: fmt-check
	for PKG in $(PACKAGES); do $(GO) test -v -cover -coverprofile $$GOPATH/src/$$PKG/coverage.txt $$PKG || exit 1; done;

html:
	$(GO) tool cover -html=coverage.txt

install:
	@hash glide > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/Masterminds/glide; \
	fi
	glide install

build: $(EXECUTABLE)

$(EXECUTABLE): $(SOURCES)
	$(GO) build -v -tags '$(TAGS)' -ldflags "$(EXTLDFLAGS)-s -w $(LDFLAGS)" -o $@

release: release-dirs release-build release-copy release-check

release-dirs:
	mkdir -p $(DIST)/binaries $(DIST)/release

release-build:
	@hash gox > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mitchellh/gox; \
	fi
	gox -os="$(TARGETS)" -arch="amd64 386" -tags="$(TAGS)" -ldflags="-s -w $(LDFLAGS)" -output="$(DIST)/binaries/$(EXECUTABLE)-$(VERSION)-{{.OS}}-{{.Arch}}"

release-copy:
	$(foreach file,$(wildcard $(DIST)/binaries/$(EXECUTABLE)-*),cp $(file) $(DIST)/release/$(notdir $(file));)

release-check:
	cd $(DIST)/release; $(foreach file,$(wildcard $(DIST)/release/$(EXECUTABLE)-*),sha256sum $(notdir $(file)) > $(notdir $(file)).sha256;)

linux_amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -a -tags '$(TAGS)' -ldflags "$(EXTLDFLAGS)-s -w $(LDFLAGS)" -o release/linux/amd64/$(EXECUTABLE)

linux_arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -a -tags '$(TAGS)' -ldflags "$(EXTLDFLAGS)-s -w $(LDFLAGS)" -o release/linux/arm64/$(EXECUTABLE)

linux_arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build -a -tags '$(TAGS)' -ldflags "$(EXTLDFLAGS)-s -w $(LDFLAGS)" -o release/arm/amd64/$(EXECUTABLE)

docker_image:
	docker build -t $(DEPLOY_ACCOUNT)/$(DEPLOY_IMAGE) .

docker_deploy:
ifeq ($(tag),)
	@echo "Usage: make $@ tag=<tag>"
	@exit 1
endif
	# deploy image
	docker tag $(DEPLOY_ACCOUNT)/$(DEPLOY_IMAGE):latest $(DEPLOY_ACCOUNT)/$(DEPLOY_IMAGE):$(tag)
	docker push $(DEPLOY_ACCOUNT)/$(DEPLOY_IMAGE):$(tag)

coverage:
	sed -i '/main.go/d' coverage.txt

clean:
	$(GO) clean -x -i ./...
	rm -rf coverage.txt $(EXECUTABLE) $(DIST) vendor

# Show source statistics.
cloc:
	@cloc -exclude-dir=vendor,node_modules .
.PHONY: cloc

version:
	@echo $(VERSION)
