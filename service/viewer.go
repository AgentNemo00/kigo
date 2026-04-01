package service

import (
	"context"
)

type Viewer struct {
	SizeX int
	SizeY int
}

func (v *Viewer) GetDimensions(ctx context.Context) (int, int, error) {
	// TODO: implement dimension retrieval from UI service
	return v.SizeX, v.SizeY, nil
}