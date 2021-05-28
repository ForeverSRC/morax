package types

import (
	"net"
	"sync"
)

type AbstractService struct {
	InShutdown AtomicBool
	Mu         sync.Mutex //protect listeners codecs
	Listeners  map[*net.Listener]struct{}
}

func (ms *AbstractService) InShuttingDown() bool {
	return ms.InShutdown.IsSet()
}

func (ms *AbstractService) TrackListener(ln *net.Listener, add bool) bool {
	ms.Mu.Lock()
	defer ms.Mu.Unlock()
	if ms.Listeners == nil {
		ms.Listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if ms.InShuttingDown() {
			return false
		}
		ms.Listeners[ln] = struct{}{}
	} else {
		delete(ms.Listeners, ln)
	}
	return true
}

func (ms *AbstractService) CloseListenersLocked() error {
	var err error
	for ln := range ms.Listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

func (ms *AbstractService) NumListeners() int {
	ms.Mu.Lock()
	defer ms.Mu.Unlock()
	return len(ms.Listeners)
}
