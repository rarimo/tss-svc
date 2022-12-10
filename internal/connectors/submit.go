package connectors

import (
	"context"
	"errors"
	"sync"
	"time"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
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
	secret   *secret.TssSecret
	clients  map[string]*con
}

func NewSubmitConnector(secret *secret.TssSecret) *SubmitConnector {
	c := &SubmitConnector{
		isClosed: false,
		secret:   secret,
		clients:  make(map[string]*con),
	}

	return c
}

func (s *SubmitConnector) Close() error {
	s.isClosed = true
	s.cleanAll()
	return nil
}

func (s *SubmitConnector) Submit(ctx context.Context, party rarimo.Party, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	if err := s.secret.Sign(request); err != nil {
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

func (s *SubmitConnector) cleanAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clients {
		c.client.Close()
	}
}

func (s *SubmitConnector) closed() error {
	if s.isClosed {
		return ErrorConnectorClosed
	}
	return nil
}
