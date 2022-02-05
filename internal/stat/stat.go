package stat

import (
	"time"
)

type Stat map[string]time.Time

func (fc Stat) Updated(next Stat) (changed, existing []string) {
	maxLen := len(fc)
	if len(next) > maxLen {
		maxLen = len(next)
	}

	changed = make([]string, 0, maxLen)
	existing = make([]string, 0, maxLen)
	for k, v := range next {
		prevT := fc[k]
		if v.Equal(prevT) {
			existing = append(existing, k)
		} else if prevT.IsZero() || !v.Equal(prevT) {
			changed = append(changed, k)
		}
	}
	return
}

func (fc Stat) Removed(next Stat) (removed []string) {
	maxLen := len(fc)
	if len(next) > maxLen {
		maxLen = len(next)
	}

	removed = make([]string, 0, maxLen)
	for k := range fc {
		nextT := next[k]
		if nextT.IsZero() {
			removed = append(removed, k)
		}
	}
	return
}
