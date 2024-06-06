package periodic_checker

import (
	"sync"

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
	var wg sync.WaitGroup
	communicator := tasks.NewCommunicator(s.cfg.Queue.AmqpURL, s.log)
	sender, err := communicator.NewSender()
	if err != nil {
		return err
	}
	callbackManager := tasks.NewCallbackManager(sender, s.log)
	wg.Add(2)
	go func() {
		defer wg.Done()
		repoTester := tasks.NewRepoTester(s.log, s.store)
		repoTester.TestRepositories()
	}()
	go func() {
		defer wg.Done()
		resourceSync := tasks.NewResourceSync(callbackManager, s.store, s.log)
		resourceSync.Poll()
	}()
	wg.Wait()
	communicator.Stop()
	return nil
}
