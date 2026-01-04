package types

import (
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/stretchr/testify/suite"
)

type MarkTestSuite struct {
	suite.Suite
}

func TestMarkSuite(t *testing.T) {
	suite.Run(t, new(MarkTestSuite))
}

func (suite *MarkTestSuite) TestMarkShapeConstants() {
	suite.Equal(MarkShape("circle"), MarkShapeCircle)
	suite.Equal(MarkShape("square"), MarkShapeSquare)
	suite.Equal(MarkShape("triangle"), MarkShapeTriangle)
}

func (suite *MarkTestSuite) TestMarkShapeAsString() {
	suite.Equal("circle", string(MarkShapeCircle))
	suite.Equal("square", string(MarkShapeSquare))
	suite.Equal("triangle", string(MarkShapeTriangle))
}

func (suite *MarkTestSuite) TestMarkStruct() {
	signal := Signal{
		Time:      time.Now(),
		Type:      SignalTypeBuyLong,
		Name:      "Test Signal",
		Reason:    "Testing",
		Symbol:    "AAPL",
		Indicator: IndicatorTypeRSI,
	}

	mark := Mark{
		MarketDataId: "md-123",
		Color:        "#FF0000",
		Shape:        MarkShapeCircle,
		Title:        "Buy Signal",
		Message:      "RSI indicates oversold condition",
		Category:     "entry",
		Signal:       optional.Some(signal),
	}

	suite.Equal("md-123", mark.MarketDataId)
	suite.Equal(MarkColor("#FF0000"), mark.Color)
	suite.Equal(MarkShapeCircle, mark.Shape)
	suite.Equal("Buy Signal", mark.Title)
	suite.Equal("RSI indicates oversold condition", mark.Message)
	suite.Equal("entry", mark.Category)
	suite.True(mark.Signal.IsSome())
	suite.Equal(signal, mark.Signal.Unwrap())
}

func (suite *MarkTestSuite) TestMarkZeroValues() {
	mark := Mark{}

	suite.Empty(mark.MarketDataId)
	suite.Empty(mark.Color)
	suite.Empty(string(mark.Shape))
	suite.Empty(mark.Title)
	suite.Empty(mark.Message)
	suite.Empty(mark.Category)
	suite.True(mark.Signal.IsNone())
}

func (suite *MarkTestSuite) TestMarkWithoutSignal() {
	mark := Mark{
		MarketDataId: "md-456",
		Color:        "#00FF00",
		Shape:        MarkShapeSquare,
		Title:        "Exit Point",
		Message:      "Take profit reached",
		Category:     "exit",
		Signal:       optional.None[Signal](),
	}

	suite.True(mark.Signal.IsNone())
	suite.Equal(MarkShapeSquare, mark.Shape)
}

func (suite *MarkTestSuite) TestMarkShapes() {
	shapes := []MarkShape{
		MarkShapeCircle,
		MarkShapeSquare,
		MarkShapeTriangle,
	}

	for _, shape := range shapes {
		mark := Mark{
			MarketDataId: "md-test",
			Shape:        shape,
		}
		suite.Equal(shape, mark.Shape)
	}
}

func (suite *MarkTestSuite) TestMarkColors() {
	colors := []MarkColor{
		MarkColorRed,
		MarkColorGreen,
		MarkColorBlue,
		MarkColorYellow,
		MarkColorPurple,
		MarkColorOrange,
	}

	for _, color := range colors {
		mark := Mark{
			Color: color,
		}
		suite.Equal(color, mark.Color)
	}
}

func (suite *MarkTestSuite) TestMarkCategories() {
	categories := []string{
		"entry",
		"exit",
		"stop_loss",
		"take_profit",
		"signal",
		"warning",
	}

	for _, category := range categories {
		mark := Mark{
			Category: category,
		}
		suite.Equal(category, mark.Category)
	}
}

func (suite *MarkTestSuite) TestMarkWithDifferentSignalTypes() {
	signalTypes := []SignalType{
		SignalTypeBuyLong,
		SignalTypeSellLong,
		SignalTypeBuyShort,
		SignalTypeSellShort,
		SignalTypeNoAction,
		SignalTypeClosePosition,
	}

	for _, sigType := range signalTypes {
		signal := Signal{
			Type: sigType,
		}
		mark := Mark{
			Signal: optional.Some(signal),
		}

		suite.True(mark.Signal.IsSome())
		suite.Equal(sigType, mark.Signal.Unwrap().Type)
	}
}

func (suite *MarkTestSuite) TestMarkShapeUniqueness() {
	shapes := []MarkShape{
		MarkShapeCircle,
		MarkShapeSquare,
		MarkShapeTriangle,
	}

	seen := make(map[MarkShape]bool)
	for _, shape := range shapes {
		suite.False(seen[shape], "Duplicate shape found: %s", shape)
		seen[shape] = true
	}
}

func (suite *MarkTestSuite) TestMarkShapeEquality() {
	shape1 := MarkShapeCircle
	shape2 := MarkShape("circle")

	suite.Equal(shape1, shape2)
}

func (suite *MarkTestSuite) TestMarkShapeInequality() {
	suite.NotEqual(MarkShapeCircle, MarkShapeSquare)
	suite.NotEqual(MarkShapeSquare, MarkShapeTriangle)
	suite.NotEqual(MarkShapeCircle, MarkShapeTriangle)
}
