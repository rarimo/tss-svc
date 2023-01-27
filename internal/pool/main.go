package pool

import (
	"context"
	"errors"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"google.golang.org/grpc"
)

const poolSz = 10000

var (
	// ErrOpShouldBeApproved appears when someone tries to add operation that has been already signed
	ErrOpShouldBeApproved = errors.New("operation should be approved")
)

// Pool implements singleton pattern
var pool *Pool

// Pool represents the pool of operation to be signed by tss protocol.
// It should take care about collecting validated state with unsigned operations only.
type Pool struct {
	rarimo *grpc.ClientConn
	log    *logan.Entry
	mu     sync.Mutex
	// Stores the order of operations to be included to the next sign,
	// but without actual information about signed status.
	rawOrder chan string
	index    map[string]struct{}
}

// NewPool returns new Pool but only once because Pool implements the singleton pattern for simple usage as
// the same instance in all injections.
func NewPool(cfg config.Config) *Pool {
	if pool == nil {
		pool = &Pool{
			rarimo:   cfg.Cosmos(),
			log:      cfg.Log(),
			rawOrder: make(chan string, poolSz),
			index:    make(map[string]struct{}),
		}
	}

	return pool
}

// Add will add operation index to the pool with signed flag check.
// Returns an error if signed check fails (cause or rpc errors).
func (p *Pool) Add(id string) error {
	if err := p.checkStatus(id); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.index[id]; !ok {
		p.index[id] = struct{}{}
		p.rawOrder <- id
	}
	return nil
}

// GetNext returns checked pool of maximum n unsigned operations or an error in case of rpc call errors.
func (p *Pool) GetNext(n uint) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	res := make([]string, 0, n)
	collected := uint(0)

	for collected < n {
		select {
		case id := <-p.rawOrder:
			err := p.checkStatus(id)
			switch err {
			case ErrOpShouldBeApproved:
				delete(p.index, id)
				continue
			case nil:
				delete(p.index, id)
				res = append(res, id)
				collected++
			default:
				p.log.WithError(err).Error("[Pool] Error querying operation")
				p.rawOrder <- id
				return res, nil
			}
		default:
			return res, nil
		}
	}

	return res, nil
}

func (p *Pool) checkStatus(id string) error {
	resp, err := rarimo.NewQueryClient(p.rarimo).Operation(context.TODO(), &rarimo.QueryGetOperationRequest{Index: id})
	if err != nil {
		return err
	}

	if resp.Operation.Status != rarimo.OpStatus_APPROVED {
		return ErrOpShouldBeApproved
	}

	return nil
}
