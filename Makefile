TARGET_FILES := \
	pcdeditor.esm.js \
	pcdeditor.wasm \
	wasm_exec.js \
	ReactPCDEditor/index.js

GO_VERSION       := $(shell go version | sed 's/^go version go\([0-9]\+\.[0-9]\+\)\.[0-9]\+\S* .*$$/\1/g')
GO_BASE_URL      := https://raw.githubusercontent.com/golang/go/refs/heads

.PHONY: serve
serve: pcdeditor.wasm wasm_exec.js pcdeditor.esm.js
	go run ./examples/serve

pcdeditor.esm.js: pcdeditor.js
	sed 's/module\.exports = /export default /' $< > $@

ReactPCDEditor/index.js: ReactPCDEditor/index.tsx package.json tsconfig.json
	pnpm tsc

pcdeditor.wasm: *.go go.*
	GOOS=js GOARCH=wasm go build \
			 -ldflags="-s -w -X 'main.Version=$(shell git rev-parse --short HEAD)' -X 'main.BuildDate=$(shell git show -s --format=%ci HEAD)'" -o $@ .

wasm_exec.js:
	wget -q $(GO_BASE_URL)/release-branch.go$(GO_VERSION)/lib/wasm/wasm_exec.js \
		|| wget -q $(GO_BASE_URL)/release-branch.go$(GO_VERSION)/misc/wasm/wasm_exec.js

.PHONY: target-files
target-files: $(TARGET_FILES)

.PHONY: pack
pack: target-files
	pnpm pack

.PHONY: version
version-$(VERSION):
	pnpm version $(VERSION) --allow-same-version --no-git-tag-version

.PHONY: publish
publish-$(VERSION):
	pnpm publish seqsense-pcdeditor-$(VERSION).tgz
