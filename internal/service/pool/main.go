package pool

import (
	"context"
	"errors"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"google.golang.org/grpc"
)

const poolSz = 1000

var (
	// ErrOpAlreadySigned appears when someone tries to add operation that has been already signed
	ErrOpAlreadySigned = errors.New("operation already signed")
)

// Pool implements singleton pattern
var pool *Pool

// Pool represents the pool of operation to be signed by tss protocol.
// It should take care about collecting validated state with unsigned operations only.
type Pool struct {
	rarimo *grpc.ClientConn
	log    *logan.Entry
	// Stores the order of operations to be included to the next sign,
	// but without actual information about signed status.
	// Also, no checks for duplication will be performed.
	rawOrder chan string
}

// NewPool returns new Pool but only once because Pool implements the singleton pattern for simple usage as
// the same instance in all injections.
func NewPool(cfg config.Config) *Pool {
	if pool == nil {
		pool = &Pool{
			rarimo:   cfg.Cosmos(),
			log:      cfg.Log(),
			rawOrder: make(chan string, poolSz),
		}
	}

	return pool
}

// Add will add operation index to the pool with signed flag check.
// Returns an error if signed check fails (cause or rpc errors).
func (p *Pool) Add(id string) error {
	if err := p.checkUnsigned(id); err != nil {
		return err
	}

	p.rawOrder <- id
	return nil
}

// GetNext returns checked pool of maximum n unsigned operations or an error in case of rpc call errors.
func (p *Pool) GetNext(n uint) ([]string, error) {
	res := make([]string, 0, n)
	collected := uint(0)

	for collected < n {
		select {
		case id := <-p.rawOrder:
			err := p.checkUnsigned(id)
			switch err {
			case ErrOpAlreadySigned:
				continue
			case nil:
				res = append(res, id)
			default:
				p.log.WithError(err).Error("error querying operation")
				p.rawOrder <- id
			}
		default:
			break
		}
	}

	return res, nil
}

func (p *Pool) checkUnsigned(id string) error {
	resp, err := rarimo.NewQueryClient(p.rarimo).Operation(context.TODO(), &rarimo.QueryGetOperationRequest{Index: id})
	if err != nil {
		return err
	}

	if resp.Operation.Signed {
		return ErrOpAlreadySigned
	}

	return nil
}
