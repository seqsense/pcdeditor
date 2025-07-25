package main

import (
	"syscall/js"

	"gopkg.in/yaml.v3"

	"github.com/seqsense/pcdeditor/blob"
)

type mapIOImpl struct{}

func (*mapIOImpl) readMap(yamlBlob, img interface{}) (*occupancyGrid, mapImage, error) {
	bj, err := blob.JS(yamlBlob)
	if err != nil {
		return nil, nil, err
	}
	r, err := bj.Reader()
	if err != nil {
		return nil, nil, err
	}
	dec := yaml.NewDecoder(r)
	m := &occupancyGrid{}
	if err := dec.Decode(m); err != nil {
		return nil, nil, err
	}
	return m, mapImageImpl(img.(js.Value)), err
}

type mapImageImpl js.Value

func (m mapImageImpl) Width() int {
	return js.Value(m).Get("width").Int()
}

func (m mapImageImpl) Height() int {
	return js.Value(m).Get("height").Int()
}

func (m mapImageImpl) Interface() interface{} {
	return js.Value(m)
}
