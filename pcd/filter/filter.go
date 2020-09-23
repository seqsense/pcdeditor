package filter

import (
	"github.com/seqsense/pcdeditor/pcd"
)

type Filter interface {
	Filter(*pcd.PointCloud) (*pcd.PointCloud, error)
}
