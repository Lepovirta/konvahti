package retry

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

type BackoffGen func(int) time.Duration

func ExponentialBackoff(min, max time.Duration) BackoffGen {
	return func(attemptNr int) time.Duration {
		mult := math.Pow(2, float64(attemptNr)) * float64(min)
		sleep := time.Duration(mult)
		if float64(sleep) != mult || sleep > max {
			sleep = max
		}
		return sleep
	}
}

type Retryable func(context.Context) error

func Retry(ctx context.Context, maxAttempts int, backoffGen BackoffGen, f Retryable) error {
	logger := log.Ctx(ctx)
	var err error
	timer := time.NewTimer(0)

	for attempt := 0; attempt < maxAttempts; attempt += 1 {
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("cancelled by context")
		case <-timer.C:
			err = f(ctx)
			if err == nil {
				return nil
			}
			delay := backoffGen(attempt)
			logger.Debug().
				Err(err).
				Str("nextAttemptIn", delay.String()).
				Int("attempt", attempt).
				Msg("retryable function failed")
			timer = time.NewTimer(delay)
		}
	}
	return err
}
