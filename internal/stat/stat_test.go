package stat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	now = time.Now()
	c1 = Stat{
		"1.txt": now.Add(time.Hour * 1),
		"2.txt": now.Add(time.Hour * 2),
		"3.txt": now.Add(time.Hour * 3),
	}
	c2 = Stat{
		"1.txt": now.Add(time.Hour * 1),
		"2.txt": now.Add(time.Hour * 22),
		"4.txt": now.Add(time.Hour * 4),
	}
)

func TestUpdated(t *testing.T) {
	u1, e1 := c1.Updated(c2)
	u2, e2 := c2.Updated(c1)

	assert.Equal(t, []string{
		"2.txt",
		"4.txt",
	}, u1)
	assert.Equal(t, []string{
		"1.txt",
	}, e1)
	assert.Equal(t, []string{
		"2.txt",
		"3.txt",
	}, u2)
	assert.Equal(t, []string{
		"1.txt",
	}, e2)
}

func TestRemoved(t *testing.T) {
	assert.Equal(t, []string{
		"3.txt",
	}, c1.Removed(c2))
	assert.Equal(t, []string{
		"4.txt",
	}, c2.Removed(c1))
}
