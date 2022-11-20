package core

import (
	"context"
	goerr "errors"
	"fmt"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3/errors"
	merkle "gitlab.com/rarify-protocol/go-merkle"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrSenderIsNotProposer = goerr.New("party is not proposer")
	ErrUnsupportedContent  = goerr.New("unsupported content")
	ErrInvalidRequestType  = goerr.New("invalid request type")
)

const MaxPoolSize = 32

type ProposalController struct {
	*bounds
	*defaultController

	mu *sync.Mutex
	wg *sync.WaitGroup

	sessionId uint64
	proposer  rarimo.Party
	result    types.ProposalData
	pool      *pool.Pool
	factory   *ControllerFactory
}

func NewProposalController(
	sessionId uint64,
	proposer rarimo.Party,
	pool *pool.Pool,
	defaultController *defaultController,
	bounds *bounds,
	factory *ControllerFactory,
) IController {
	return &ProposalController{
		bounds:            bounds,
		defaultController: defaultController,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		sessionId:         sessionId,
		proposer:          proposer,
		pool:              pool,
		factory:           factory,
	}
}

var _ IController = &ProposalController{}

func (p *ProposalController) Receive(request *types.MsgSubmitRequest) error {
	if err := p.checkSender(request); err != nil {
		return err
	}

	if request.Type != types.RequestType_Proposal {
		return ErrInvalidRequestType
	}

	proposal := new(types.ProposalRequest)
	if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if proposal.Session == p.sessionId && p.validate(proposal.Indexes, proposal.Root) {
		if len(proposal.Indexes) == 0 {
			p.infof("Received empty pool. Skipping.")
		}

		p.finish(proposal.Root, proposal.Indexes)
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
	p.mu.Lock()
	defer p.mu.Unlock()

	abounds := NewBounds(p.End()+1, p.params.Step(AcceptingIndex).Duration)
	if len(p.result.Indexes) == 0 {
		sBounds := NewBounds(abounds.End()+1, p.params.Step(SigningIndex).Duration)
		fBounds := NewBounds(sBounds.End()+1, p.params.Step(FinishingIndex).Duration)
		finish := p.factory.GetFinishController(p.sessionId, types.SignatureData{}, fBounds)
		signing := p.factory.GetEmptyController(finish, sBounds)
		acceptance := p.factory.GetEmptyController(signing, abounds)
		return acceptance
	}

	return p.factory.GetAcceptanceController(p.sessionId, p.result, abounds)
}

func (p *ProposalController) finish(root string, indexes []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.infof("Pool root: %s", root)
	p.infof("Indexes: %v", indexes)
	p.result = types.ProposalData{Indexes: indexes, Root: root}
}

func (p *ProposalController) checkSender(request *types.MsgSubmitRequest) error {
	sender, err := p.auth.Auth(request)
	if err != nil {
		return err
	}

	if !Equal(&sender, &p.proposer) {
		return ErrSenderIsNotProposer
	}

	return nil
}

func (p *ProposalController) validate(indexes []string, root string) bool {
	if len(indexes) == 0 {
		return true
	}

	contents, err := p.getContents(context.Background(), indexes)
	if err != nil {
		p.errorf(err, "error preparing contents")
		return false
	}

	return hexutil.Encode(merkle.NewTree(crypto.Keccak256, contents...).Root()) == root
}

func (p *ProposalController) run(ctx context.Context) {
	defer func() {
		p.infof("Controller finished")
		p.wg.Done()
	}()

	if p.proposer.PubKey != p.secret.ECDSAPubKeyStr() {
		p.infof("Proposer is another party")
		return
	}

	ids, root, err := p.getNewPool(ctx)
	if err != nil {
		p.errorf(err, "Error preparing pool to propose")
		return
	}

	p.finish(root, ids)
	if len(ids) == 0 {
		p.infof("Empty pool. Skipping.")
		return
	}

	details, err := cosmostypes.NewAnyWithValue(&types.ProposalRequest{Session: p.sessionId, Indexes: ids, Root: root})
	if err != nil {
		p.errorf(err, "Error parsing details")
		return
	}

	p.SubmitAll(ctx, &types.MsgSubmitRequest{
		Type:        types.RequestType_Proposal,
		IsBroadcast: true,
		Details:     details,
	})
}

func (p *ProposalController) getNewPool(ctx context.Context) ([]string, string, error) {
	ids, err := p.pool.GetNext(MaxPoolSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "error preparing pool")
	}

	if len(ids) == 0 {
		return []string{}, "", nil
	}

	contents, err := p.getContents(ctx, ids)
	if err != nil {
		return nil, "", err
	}

	return ids, hexutil.Encode(merkle.NewTree(crypto.Keccak256, contents...).Root()), nil
}

func (p *ProposalController) getContents(ctx context.Context, ids []string) ([]merkle.Content, error) {
	contents := make([]merkle.Content, 0, len(ids))

	for _, id := range ids {
		resp, err := rarimo.NewQueryClient(p.rarimo).Operation(context.TODO(), &rarimo.QueryGetOperationRequest{Index: id})
		if err != nil {
			return nil, errors.Wrap(err, "error fetching operation")
		}

		if resp.Operation.Signed {
			return nil, pool.ErrOpAlreadySigned
		}

		switch resp.Operation.OperationType {
		case rarimo.OpType_TRANSFER:
			content, err := p.getTransferContent(ctx, resp.Operation)
			if err != nil {
				return nil, err
			}
			contents = append(contents, content)
		default:
			return nil, ErrUnsupportedContent
		}
	}

	return contents, nil
}

func (p *ProposalController) getTransferContent(ctx context.Context, op rarimo.Operation) (merkle.Content, error) {
	transfer, err := pkg.GetTransfer(op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	infoResp, err := token.NewQueryClient(p.rarimo).Info(ctx, &token.QueryGetInfoRequest{Index: transfer.TokenIndex})
	if err != nil {
		return nil, errors.Wrap(err, "error getting token info entry")
	}

	itemResp, err := token.NewQueryClient(p.rarimo).Item(ctx, &token.QueryGetItemRequest{
		TokenAddress: infoResp.Info.Chains[transfer.ToChain].TokenAddress,
		TokenId:      infoResp.Info.Chains[transfer.ToChain].TokenId,
		Chain:        transfer.ToChain,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error getting token item entry")
	}

	content, err := pkg.GetTransferContent(&itemResp.Item, p.params.ChainParams(transfer.ToChain), transfer)
	return content, errors.Wrap(err, "error creating content")
}

func (p *ProposalController) infof(msg string, args ...interface{}) {
	p.Infof("[Proposal %d] - %s", p.sessionId, fmt.Sprintf(msg, args))
}

func (p *ProposalController) errorf(err error, msg string, args ...interface{}) {
	p.WithError(err).Errorf("[Proposal %d] - %s", p.sessionId, fmt.Sprintf(msg, args))
}
