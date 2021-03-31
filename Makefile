GOROOT := $(shell go env GOROOT)
RELEASE_FILES := \
	iframe.html \
	index.html \
	pcdeditor.css \
	pcdeditor.js \
	pcdeditor.wasm \
	wasm_exec.js

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
release: $(addprefix release/,$(RELEASE_FILES))

release/%: %
	mkdir -p release
	cp $< $@
