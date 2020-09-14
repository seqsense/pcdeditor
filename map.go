package main

type occupancyGrid struct {
	Image          string    `yaml:"image"`
	Resolution     float32   `yaml:"resolution"`
	Origin         []float32 `yaml:"origin"`
	Height         float32   `yaml:"height"`
	Negate         int       `yaml:"nagate"`
	OccupiedThresh float32   `yaml:"occupied_thresh"`
	FreeThresh     float32   `yaml:"free_thresh"`
}

type mapImage interface {
	Width() int
	Height() int
	Interface() interface{}
}
