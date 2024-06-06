package worker_server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/flightctl/flightctl/internal/config"
	"github.com/flightctl/flightctl/internal/store"
	"github.com/flightctl/flightctl/internal/tasks"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg   *config.Config
	log   logrus.FieldLogger
	store store.Store
}

// New returns a new instance of a flightctl server.
func New(
	cfg *config.Config,
	log logrus.FieldLogger,
	store store.Store,
) *Server {
	return &Server{
		cfg:   cfg,
		log:   log,
		store: store,
	}
}

func (s *Server) Run() error {
	s.log.Println("Initializing async jobs")
	communicator := tasks.NewCommunicator(s.cfg.Queue.AmqpURL, s.log)
	sender, err := communicator.NewSender()
	if err != nil {
		return err
	}
	callbackManager := tasks.NewCallbackManager(sender, s.log)
	receiver, err := communicator.NewReceiver()
	if err != nil {
		return err
	}
	if err = receiver.Receive(tasks.DispatchCallbacks(s.store, callbackManager)); err != nil {
		return err
	}
	sigShutdown := make(chan os.Signal, 1)
	signal.Notify(sigShutdown, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigShutdown
		s.log.Println("Shutdown signal received")
		communicator.Stop()
	}()
	communicator.Wait()

	return nil
}
