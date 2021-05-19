package utils

import (
	"math/rand"
	"time"
)
import (
	"github.com/ForeverSRC/morax/common/constants"
)

func NextPollInterval(pollIntervalBase *time.Duration) time.Duration {
	// Add 10% jitter.
	interval := *pollIntervalBase + time.Duration(rand.Intn(int(*pollIntervalBase/10)))
	// Double and clamp for next time.
	*pollIntervalBase *= 2
	if *pollIntervalBase > constants.ShutdownPollIntervalMax {
		*pollIntervalBase = constants.ShutdownPollIntervalMax
	}
	return interval
}
