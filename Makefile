GOROOT := $(shell go env GOROOT)

.PHONY: serve
serve: pcdeditor.wasm wasm_exec.js data/map.pcd
	go run ./examples/serve

pcdeditor.wasm: *.go */*.go
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $@ .

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@
