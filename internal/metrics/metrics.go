package metrics

import (
	"errors"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

var statsdClient *statsd.Client

func InitStatsdClient(addr string) (*statsd.Client, error) {
	// if the addr is empty, statsd will look at the envvar DD_DOGSTATSD_URL
	var err error
	s, err := statsd.New(addr,
		statsd.WithNamespace("blocklake."),
		statsd.WithBufferFlushInterval(time.Second*2),
	)

	statsdClient = s

	return s, err
}

func GetStatsdClient() *statsd.Client {
	if statsdClient == nil {
		panic(errors.New("statsd client not initialized"))
	}
	return statsdClient
}
