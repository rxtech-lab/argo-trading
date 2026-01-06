package types

import "github.com/moznion/go-optional"

type MarkShape string

const (
	MarkShapeCircle   MarkShape = "circle"
	MarkShapeSquare   MarkShape = "square"
	MarkShapeTriangle MarkShape = "triangle"
)

type MarkColor string

const (
	MarkColorRed    MarkColor = "red"
	MarkColorGreen  MarkColor = "green"
	MarkColorBlue   MarkColor = "blue"
	MarkColorYellow MarkColor = "yellow"
	MarkColorPurple MarkColor = "purple"
	MarkColorOrange MarkColor = "orange"
)

type MarkLevel string

const (
	MarkLevelInfo    MarkLevel = "info"
	MarkLevelWarning MarkLevel = "warning"
	MarkLevelError   MarkLevel = "error"
)

type Mark struct {
	MarketDataId string
	Color        MarkColor
	Shape        MarkShape
	Level        MarkLevel
	Title        string
	Message      string
	Category     string
	Signal       optional.Option[Signal]
}
