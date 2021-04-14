GOROOT := $(shell go env GOROOT)
TARGET_FILES := \
	pcdeditor.esm.js \
	pcdeditor.wasm \
	wasm_exec.js \
	ReactPCDEditor/index.js

.PHONY: serve
serve: pcdeditor.wasm wasm_exec.js pcdeditor.esm.js
	go run ./examples/serve

pcdeditor.esm.js: pcdeditor.js
	sed 's/module\.exports = /export default /' $< > $@

ReactPCDEditor/index.js: ReactPCDEditor/index.tsx package.json tsconfig.json
	npm run-script tsc

pcdeditor.wasm: *.go go.*
	GOOS=js GOARCH=wasm go build \
			 -ldflags="-s -w -X 'main.Version=$(shell git rev-parse --short HEAD)' -X 'main.BuildDate=$(shell git show -s --format=%ci HEAD)'" -o $@ .

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@

.PHONY: target-files
target-files: $(TARGET_FILES)

.PHONY: pack
pack: target-files
	npm pack

.PHONY: version
version-$(VERSION):
	npm version $(VERSION) --allow-same-version --no-git-tag-version

.PHONY: publish
publish-$(VERSION):
	npm publish seqsense-pcdeditor-$(VERSION).tgz
