package connectors

import (
	"context"
	"errors"
	"sync"
	"time"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const (
	canCloseAfter = 1 * time.Hour
	cleanRetry    = 1 * time.Hour
)

var ErrorConnectorClosed = errors.New("connector already closed")

type con struct {
	client   *grpc.ClientConn
	lastUsed time.Time
}

// SubmitConnector submits signed requests to the party.
// Also holds buffer of connections to reduce submitting time.
type SubmitConnector struct {
	mu       sync.Mutex
	isClosed bool
	secret   *local.Secret
	clients  map[string]*con
}

func NewSubmitConnector(cfg config.Config) *SubmitConnector {
	c := &SubmitConnector{
		isClosed: false,
		secret:   local.NewSecret(cfg),
		clients:  make(map[string]*con),
	}

	go c.runCleaner()
	return c
}

func (s *SubmitConnector) Close() error {
	s.isClosed = true
	return nil
}

func (s *SubmitConnector) Submit(ctx context.Context, party rarimo.Party, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	err := s.secret.SignRequest(request)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.getClient(party.Address)
	if err != nil {
		return nil, err
	}

	return types.NewServiceClient(client).Submit(ctx, request)
}

func (s *SubmitConnector) getClient(addr string) (*grpc.ClientConn, error) {
	if err := s.closed(); err != nil {
		return nil, err
	}

	if client, ok := s.clients[addr]; ok && client != nil {
		client.lastUsed = time.Now().UTC()
		return client.client, nil
	}

	client, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	s.clients[addr] = &con{
		client:   client,
		lastUsed: time.Now().UTC(),
	}

	return client, nil
}

func (s *SubmitConnector) runCleaner() {
	for {
		time.Sleep(cleanRetry)
		if err := s.closed(); err != nil {
			s.cleanAll()
			return
		}

		s.clean()
	}
}

func (s *SubmitConnector) cleanAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clients {
		c.client.Close()
	}
}

func (s *SubmitConnector) clean() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for addr, c := range s.clients {
		if c.lastUsed.Before(time.Now().UTC().Add(-canCloseAfter)) {
			c.client.Close()
			s.clients[addr] = nil
		}
	}
}

func (s *SubmitConnector) closed() error {
	if s.isClosed {
		return ErrorConnectorClosed
	}
	return nil
}
