package swiftargo

// StringCollection is a gomobile-compatible interface for passing string collections from Swift.
// Note: gomobile doesn't support returning slices, so we use Get(i) + Size() pattern.
type StringCollection interface {
	Add(s string) StringCollection
	Get(i int) string
	Size() int
}

type StringArray struct {
	items []string
}

// NewStringArray creates a new empty StringArray.
// This is the constructor that should be used from Swift via gomobile.
func NewStringArray() *StringArray {
	return &StringArray{items: []string{}}
}

func (a *StringArray) Add(s string) StringCollection {
	a.items = append(a.items, s)

	return a
}

func (a *StringArray) Get(i int) string {
	if i < 0 || i >= len(a.items) {
		return ""
	}

	return a.items[i]
}

func (a *StringArray) Size() int {
	return len(a.items)
}

func (a *StringArray) All() []string {
	return a.items
}
