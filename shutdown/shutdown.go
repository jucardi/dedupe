package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"github.com/jucardi/go-logger-lib/log"
)

type WithContextHookFunc func(ctx context.Context) error
type WithoutContextHookFunc func() error

type hook struct {
	contextHookFunc WithContextHookFunc
	hookFunc        WithoutContextHookFunc
}

var hooks []hook

// ListenForSignals for a TERM or INT signal.  Once the signal is caught all shutdown hooks will be
// executed allowing a graceful shutdown
func ListenForSignals() {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit

		log.Infof("signal captured: %s", sig.String())
		log.Infof("hooks: %+v", hooks)
		invokeHooks()
		os.Exit(0)
	}()
}

// AddShutdownHook associates a no-arg func to be called when a signal is caught allowing for
// cleanup.
//
// Note: the function should not block for a period of time and hold up the shutdown
// of this app/service. All shutdown hooks must be independent of each other
// since they are executed concurrently for faster shutdown.
func AddShutdownHook(f WithoutContextHookFunc) {
	hooks = append(hooks, hook{hookFunc: f})
}

// AddShutdownHook associates a func that accepts a context.Context argument. The function will
// be called when a signal is caught allowing for cleanup.
//
// Note: The context has a 5 second time limit for protection on shutdown.  All shutdown hooks must be
// independent of each other since they are executed concurrently for faster shutdown.
func AddContextShutdownHook(f WithContextHookFunc) {
	hooks = append(hooks, hook{contextHookFunc: f})
}

func invokeHooks() {
	var wg sync.WaitGroup

	wg.Add(len(hooks))

	for _, hook := range hooks {
		go func() {
			defer wg.Done()
			hook.execute()
		}()
	}

	wg.Wait()
}

func (h *hook) execute() {
	var err error
	if h.contextHookFunc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = h.contextHookFunc(ctx)
	} else {
		err = h.hookFunc()
	}
	if err != nil {
		log.Warn(err.Error())
	}
}
