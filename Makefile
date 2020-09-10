GOROOT := $(shell go env GOROOT)

.PHONY: serve
serve: pcdeditor.wasm wasm_exec.js data/map.pcd
	go run ./examples/serve

pcdeditor.wasm: *.go */*.go
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $@ .

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@

.PHONY: release
release: release/pcdeditor.wasm release/wasm_exec.js release/index.html

release/pcdeditor.wasm: pcdeditor.wasm
	mkdir -p release
	cp $< $@

release/wasm_exec.js: wasm_exec.js
	mkdir -p release
	cp $< $@

release/index.html: index.html
	mkdir -p release
	cp $< $@
