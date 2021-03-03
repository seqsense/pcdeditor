GOROOT := $(shell go env GOROOT)

.PHONY: serve
serve: pcdeditor.wasm wasm_exec.js fixture/map.pcd vendor_js/postmate.min.js
	go run ./examples/serve

vendor_js/postmate.min.js:
	mkdir -p vendor_js
	wget -O $@ https://cdn.jsdelivr.net/npm/postmate@1.5.2/build/postmate.min.js

pcdeditor.wasm: *.go */*.go
	GOOS=js GOARCH=wasm go build \
			 -ldflags="-s -w -X 'main.Version=$(shell git rev-parse --short HEAD)' -X 'main.BuildDate=$(shell git show -s --format=%ci HEAD)'" -o $@ .

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@

.PHONY: release
release: release/pcdeditor.wasm release/wasm_exec.js release/index.html release/iframe.html

release/pcdeditor.wasm: pcdeditor.wasm
	mkdir -p release
	cp $< $@

release/wasm_exec.js: wasm_exec.js
	mkdir -p release
	cp $< $@

release/index.html: index.html
	mkdir -p release
	cp $< $@

release/iframe.html: iframe.html
	mkdir -p release
	cp $< $@
