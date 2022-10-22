package session

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

type IPool interface {
	Add(id string) error
	GetNext(n uint) []string
}

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

func NewPool(cfg config.Config) IPool {
	return &Pool{
		rarimo:   cfg.Cosmos(),
		log:      cfg.Log(),
		rawOrder: make(chan string, poolSz),
	}
}

// IPool implementation

var _ IPool = &Pool{}

func (p *Pool) Add(id string) error {
	if err := p.checkUnsigned(id); err != nil {
		return err
	}

	p.rawOrder <- id
	return nil
}

func (p *Pool) GetNext(n uint) []string {
	res := make([]string, 0, n)
	collected := uint(0)

	for collected < n {
		select {
		case id := <-p.rawOrder:
			if err := p.checkUnsigned(id); err != nil {
				continue
			}
			res = append(res, id)
		default:
			break
		}
	}

	return res
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
