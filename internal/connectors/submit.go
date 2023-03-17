package connectors

import (
	"context"
	"sync"
	"time"

	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

type con struct {
	client   *grpc.ClientConn
	lastUsed time.Time
}

var clientsBuffer = struct {
	mu      sync.Mutex
	clients map[string]*con
}{
	clients: make(map[string]*con),
}

// SubmitConnector submits signed requests to the party.
// Also holds buffer of connections to reduce submitting time.
type SubmitConnector struct {
	secret *secret.TssSecret
}

func NewSubmitConnector(secret *secret.TssSecret) *SubmitConnector {
	c := &SubmitConnector{
		secret: secret,
	}

	return c
}

func (s *SubmitConnector) Submit(ctx context.Context, party rarimo.Party, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	if err := s.secret.Sign(request); err != nil {
		return nil, err
	}

	var client *con
	var err error

	func() {
		clientsBuffer.mu.Lock()
		defer clientsBuffer.mu.Unlock()

		client, err = s.getClient(party.Address)
	}()

	if err != nil {
		return nil, err
	}

	return types.NewServiceClient(client.client).Submit(ctx, request)
}

func (s *SubmitConnector) getClient(addr string) (*con, error) {
	if client, ok := clientsBuffer.clients[addr]; ok && client != nil {
		client.lastUsed = time.Now().UTC()
		return client, nil
	}

	client, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	con := &con{
		client:   client,
		lastUsed: time.Now().UTC(),
	}

	clientsBuffer.clients[addr] = con

	return con, nil
}
