GOROOT := $(shell go env GOROOT)

.PHONY: serve
serve: pcdviewer.wasm wasm_exec.js
	go run ./examples/serve

pcdviewer.wasm: *.go */*.go
	GOOS=js GOARCH=wasm go build -o $@ .

wasm_exec.js: $(GOROOT)/misc/wasm/wasm_exec.js
	cp $< $@
