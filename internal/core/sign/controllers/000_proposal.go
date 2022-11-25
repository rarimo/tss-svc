package controllers

import (
	"context"
	"sync"

	"github.com/anyswap/FastMulThreshold-DSA/crypto"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3/errors"
	merkle "gitlab.com/rarify-protocol/go-merkle"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const MaxPoolSize = 32

type ProposalController struct {
	*defaultController
	mu *sync.Mutex
	wg *sync.WaitGroup

	bounds *core.Bounds
	data   LocalSessionData
	result LocalProposalData

	storage secret.Storage
	client  *grpc.ClientConn
	pool    *pool.Pool
	factory *ControllerFactory
}

var _ IController = &ProposalController{}

func (p *ProposalController) Receive(request *types.MsgSubmitRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	sender, err := p.auth.Auth(request)
	if err != nil {
		return err
	}

	if !core.Equal(&sender, &p.data.Proposer) {
		return ErrSenderIsNotProposer
	}

	if request.Type != types.RequestType_Proposal {
		return ErrInvalidRequestType
	}

	proposal := new(types.ProposalRequest)
	if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if proposal.Type != types.SessionType_DefaultSession {
		return ErrInvalidRequestType
	}

	data := new(types.DefaultSessionProposalData)
	if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
		return errors.Wrap(err, "error unmarshalling details")
	}

	if p.validate(data.Indexes, data.Root) {
		p.result.Indexes = data.Indexes
		p.result.Root = data.Root
	}

	return nil
}

func (p *ProposalController) Run(ctx context.Context) {
	p.wg.Add(1)
	go p.run(ctx)
}

func (p *ProposalController) WaitFor() {
	p.wg.Wait()
}

func (p *ProposalController) Next() IController {
	if len(p.result.Indexes) == 0 {
		bounds := core.NewBounds(p.bounds.Finish+1,
			p.params.Step(AcceptingIndex).Duration+
				1+p.params.Step(SigningIndex).Duration+
				1+p.params.Step(FinishingIndex).Duration,
		)

		data := LocalSignatureData{}
		data.LocalSessionData = p.data
		return p.factory.GetFinishController(bounds, data)
	}

	return p.factory.GetAcceptanceController(core.NewBounds(p.bounds.Finish+1, p.params.Step(AcceptingIndex).Duration), p.result)
}

func (p *ProposalController) Bounds() *core.Bounds {
	return p.bounds
}

func (p *ProposalController) validate(indexes []string, root string) bool {
	if len(indexes) == 0 {
		return true
	}

	ops, err := GetOperations(p.client, indexes...)
	if err != nil {
		return false
	}

	contents, err := GetContents(p.client, ops...)
	if err != nil {
		return false
	}

	return hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()) == root
}

func (p *ProposalController) run(ctx context.Context) {
	defer func() {
		p.log.Info("Proposal controller finished")
		p.wg.Done()
	}()

	if p.data.Proposer.Account != p.storage.AccountAddressStr() {
		p.log.Info("Proposer is another party")
		return
	}

	ids, root, err := p.getNewPool()
	if err != nil {
		p.log.WithError(err).Error("Error preparing pool to propose")
		return
	}

	func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.result.Indexes = ids
		p.result.Root = root
	}()

	if len(ids) == 0 {
		p.log.Info("Empty pool. Skipping.")
		return
	}

	data, err := cosmostypes.NewAnyWithValue(&types.DefaultSessionProposalData{Indexes: ids, Root: root})
	if err != nil {
		p.log.WithError(err).Error("Error parsing data")
		return
	}

	details, err := cosmostypes.NewAnyWithValue(&types.ProposalRequest{Type: types.SessionType_DefaultSession, Details: data})
	if err != nil {
		p.log.WithError(err).Error("Error parsing details")
		return
	}

	p.broadcast.SubmitAll(ctx, &types.MsgSubmitRequest{
		Id:          p.data.SessionId,
		Type:        types.RequestType_Proposal,
		IsBroadcast: true,
		Details:     details,
	})
}

func (p *ProposalController) getNewPool() ([]string, string, error) {
	ids, err := p.pool.GetNext(MaxPoolSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "error preparing pool")
	}

	if len(ids) == 0 {
		return []string{}, "", nil
	}

	ops, err := GetOperations(p.client, ids...)
	if err != nil {
		return nil, "", err
	}

	contents, err := GetContents(p.client, ops...)
	if err != nil {
		return nil, "", err
	}

	return ids, hexutil.Encode(merkle.NewTree(crypto.Keccak256, contents...).Root()), nil
}
