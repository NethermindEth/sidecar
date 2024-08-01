package shutdown

import (
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func CreateGracefulShutdownChannel() chan os.Signal {
	gracefulShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, syscall.SIGTERM, syscall.SIGINT)

	return gracefulShutdown
}

func ListenForShutdown(
	signalChan chan os.Signal,
	done chan bool,
	signalHandler func(),
	timeToWait time.Duration,
	l *zap.Logger,
) {
	sig := <-signalChan
	switch sig {
	case syscall.SIGTERM, syscall.SIGINT:
		l.Sugar().Infof("caught signal %v", sig)

		signalHandler()

		l.Sugar().Infof("Waiting %v seconds to exit...", timeToWait.Seconds())
		time.Sleep(timeToWait)

		l.Sugar().Infof("Exiting")
		close(done)
	}
}
