package retry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoffMinMax(t *testing.T) {
	minTime := time.Millisecond * 100
	maxTime := time.Millisecond * 2000
	backOffGen := ExponentialBackoff(minTime, maxTime)

	for i := 0; i < 10000; i++ {
		nextDuration := backOffGen(i)
		assert.LessOrEqual(t, minTime, nextDuration)
		assert.LessOrEqual(t, nextDuration, maxTime)
	}
}

func oneMsBackoff(attempt int) time.Duration {
	return 1 * time.Millisecond
}

type callCounter struct {
	nrOfCalls int
}

func (c *callCounter) call() {
	c.nrOfCalls += 1
}

func TestRetryInstantHappy(t *testing.T) {
	cc := &callCounter{}

	err := Retry(
		context.Background(),
		3,
		oneMsBackoff,
		func(ctx context.Context) error {
			cc.call()
			return nil
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, 1, cc.nrOfCalls)
}

func TestRetryEventualHappy(t *testing.T) {
	cc := &callCounter{}

	err := Retry(
		context.Background(),
		3,
		oneMsBackoff,
		func(ctx context.Context) error {
			cc.call()
			if cc.nrOfCalls < 2 {
				return fmt.Errorf("nr of calls = %d", cc.nrOfCalls)
			}
			return nil
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, 2, cc.nrOfCalls)
}

func TestRetryUnhappy(t *testing.T) {
	cc := &callCounter{}

	err := Retry(
		context.Background(),
		3,
		oneMsBackoff,
		func(ctx context.Context) error {
			cc.call()
			return fmt.Errorf("nr of calls = %d", cc.nrOfCalls)
		},
	)

	assert.Error(t, err)
	assert.Equal(t, 4, cc.nrOfCalls)
}

func TestRetryCancel(t *testing.T) {
	cc := &callCounter{}
	ctx, cancel := context.WithCancel(context.Background())

	err := Retry(
		ctx,
		3,
		oneMsBackoff,
		func(ctx context.Context) error {
			cc.call()
			if cc.nrOfCalls == 2 {
				cancel()
			}
			if cc.nrOfCalls > 2 {
				return nil
			}
			return fmt.Errorf("nr of calls = %d", cc.nrOfCalls)
		},
	)

	assert.Error(t, err)
	assert.Equal(t, 2, cc.nrOfCalls)
}
