package connectors

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"github.com/rarimo/tss-svc/internal/secret"
	"github.com/rarimo/tss-svc/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func (s *SubmitConnector) Submit(ctx context.Context, party *rarimo.Party, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
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

	var client *grpc.ClientConn
	var err error

	connectSecurityOptions := grpc.WithInsecure()

	if s.secret.TLS() {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
		}

		connectSecurityOptions = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	}

	client, err = grpc.Dial(addr, connectSecurityOptions)
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
