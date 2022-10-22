package party

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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

type SubmitConnector struct {
	mu       sync.Mutex
	isClosed bool
	prvKey   *ecdsa.PrivateKey
	clients  map[string]*con
}

func NewSubmitConnector(prvKey *ecdsa.PrivateKey) *SubmitConnector {
	c := &SubmitConnector{
		isClosed: false,
		prvKey:   prvKey,
		clients:  make(map[string]*con),
	}

	go c.runCleaner()
	return c
}

func (s *SubmitConnector) Close() error {
	s.isClosed = true
	return nil
}

func (s *SubmitConnector) SignAdnSubmit(ctx context.Context, addr string, request types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	data, err := request.Details.Marshal()
	if err != nil {
		return nil, err
	}

	signature, err := crypto.Sign(crypto.Keccak256(data), s.prvKey)
	if err != nil {
		return nil, err
	}

	request.Signature = hexutil.Encode(signature)

	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.getClient(addr)
	if err != nil {
		return nil, err
	}

	return types.NewServiceClient(client).Submit(ctx, &request)
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
