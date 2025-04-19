package types

import "github.com/moznion/go-optional"

type MarkShape string

const (
	MarkShapeCircle   MarkShape = "circle"
	MarkShapeSquare   MarkShape = "square"
	MarkShapeTriangle MarkShape = "triangle"
)

type Mark struct {
	MarketDataId string
	Color        string
	Shape        MarkShape
	Title        string
	Message      string
	Category     string
	Signal       optional.Option[Signal]
}
