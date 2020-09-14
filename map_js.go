package main

import (
	"errors"
	"path"
	"syscall/js"

	"gopkg.in/yaml.v2"
)

type mapIOImpl struct{}

func (*mapIOImpl) readMap(yamlPath string) (*occupancyGrid, mapImage, error) {
	b, err := fetchGet(yamlPath)
	if err != nil {
		return nil, nil, err
	}
	m := &occupancyGrid{}
	if err := yaml.Unmarshal(b, m); err != nil {
		return nil, nil, err
	}
	imgPath := path.Dir(yamlPath) + "/" + m.Image

	img := js.Global().Get("Image").New()
	chOK := make(chan bool, 1)
	img.Call("addEventListener", "load",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chOK <- true
			return nil
		}),
	)
	img.Call("addEventListener", "error",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			chOK <- false
			return nil
		}),
	)
	img.Set("src", imgPath)

	if !<-chOK {
		return nil, nil, errors.New("failed to load map image")
	}

	return m, mapImageImpl(img), err
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
