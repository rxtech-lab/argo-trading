package cache

import (
	"github.com/moznion/go-optional"
)

type Cache interface {
	Reset()
}

type RangeFilterState struct {
	Initialized bool    `json:"initialized"`
	PrevFilt    float64 `json:"prev_filt"`
	PrevSource  float64 `json:"prev_source"`
	Upward      float64 `json:"upward"`
	Downward    float64 `json:"downward"`
	Symbol      string  `json:"symbol"` // Symbol this state applies to
}

type WaddahAttarState struct {
	Initialized bool    `json:"initialized"`
	PrevMACD    float64 `json:"prev_macd"`
	PrevSignal  float64 `json:"prev_signal"`
	PrevHist    float64 `json:"prev_hist"`
	PrevATR     float64 `json:"prev_atr"`
	Symbol      string  `json:"symbol"` // Symbol this state applies to
}

type CacheV1 struct {
	RangeFilterState optional.Option[RangeFilterState]
	WaddahAttarState optional.Option[WaddahAttarState]
	otherData        map[string]any
}

func NewCacheV1() Cache {
	return &CacheV1{
		RangeFilterState: optional.None[RangeFilterState](),
		WaddahAttarState: optional.None[WaddahAttarState](),
		otherData:        make(map[string]any),
	}
}

// Reset implements cache.Cache.
func (c *CacheV1) Reset() {
	c.RangeFilterState = optional.None[RangeFilterState]()
	c.WaddahAttarState = optional.None[WaddahAttarState]()
	c.otherData = make(map[string]any)
}

// Set cache data by key. Don't use this method if you want to add a state for indicator. Modify the CacheV1 struct directly.
// This is for strategy only!
func (c *CacheV1) Set(key string, value any) {
	c.otherData[key] = value
}

// Get cache data by key. Don't use this method if you want to get a state for indicator. Use the method in the indicator struct instead.
func (c *CacheV1) Get(key string) (any, bool) {
	value, ok := c.otherData[key]
	return value, ok
}
