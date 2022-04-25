package matchmaking

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gunnermanx/simplegameserver/auth"
	"github.com/gunnermanx/simplegameserver/config"
	"github.com/gunnermanx/simplegameserver/datastore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	GRACEFUL_SHUTDOWN_TIME_S = 10
)

type SimpleMatchmakingServer struct {
	sync.Mutex

	config   *config.MatchmakingServerConfig
	serveMux *http.ServeMux
	server   *http.Server
	logger   *logrus.Logger

	datastore    datastore.Datastore
	authProvider auth.AuthProvider
}

func New(
	conf *config.MatchmakingServerConfig,
	logger *logrus.Logger,
	ap auth.AuthProvider,
	ds datastore.Datastore,
) (s *SimpleMatchmakingServer) {

	s = &SimpleMatchmakingServer{
		config:       conf,
		logger:       logger,
		authProvider: ap,
		datastore:    ds,
		serveMux:     http.NewServeMux(),
	}

	s.setupHandlers()
	s.server = &http.Server{
		Handler: s,
	}

	return
}

// Start the matchmaking server
func (sms *SimpleMatchmakingServer) Start() (err error) {
	var listener net.Listener
	// TODO remove localhost
	if listener, err = net.Listen("tcp", fmt.Sprintf(":%s", sms.config.Port)); err != nil {
		err = errors.Wrap(err, "failed to start matchmaking server")
		sms.logger.Error(err)
		return
	}

	// Start the http server
	errc := make(chan error, 1)
	go func() {
		sms.logger.Infof("Starting game server on: %s", listener.Addr().String())
		errc <- sms.server.Serve(listener)
	}()

	// Wait for termination or errors
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err = <-errc:
		sms.logger.Errorf("failed to serve: %s", err.Error())
	case sig := <-sigs:
		sms.logger.Errorf("terminating on sig: %v", sig)
	}

	// Gracefully shutdown with timeout of 10s
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*GRACEFUL_SHUTDOWN_TIME_S)
	defer cancel()
	return sms.server.Shutdown(ctx)
}
