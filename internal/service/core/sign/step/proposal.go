package step

import (
	"context"
	goerr "errors"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	merkle "gitlab.com/rarify-protocol/go-merkle"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const MaxPoolSize = 32

var ErrUnsupportedContent = goerr.New("unsupported content")

type ProposalController struct {
	wg       *sync.WaitGroup
	id       uint64
	proposer rarimo.Party

	result chan *session.Proposal

	connector *connectors.BroadcastConnector
	pool      *pool.Pool
	rarimo    *grpc.ClientConn
	params    *local.Params
	secret    *local.Secret
	log       *logan.Entry
}

func NewProposalController(
	id uint64,
	params *local.Params,
	secret *local.Secret,
	proposer rarimo.Party,
	result chan *session.Proposal,
	connector *connectors.BroadcastConnector,
	pool *pool.Pool,
	rarimo *grpc.ClientConn,
	log *logan.Entry,
) *ProposalController {
	return &ProposalController{
		wg:        &sync.WaitGroup{},
		id:        id,
		params:    params,
		secret:    secret,
		proposer:  proposer,
		result:    result,
		connector: connector,
		pool:      pool,
		rarimo:    rarimo,
		log:       log,
	}
}

var _ IController = &ProposalController{}

func (p *ProposalController) Receive(sender rarimo.Party, request types.MsgSubmitRequest) error {
	if request.Type == types.RequestType_Proposal && sender.Account == p.proposer.Account {
		proposal := new(types.ProposalRequest)

		if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
			return err
		}

		if proposal.Session == p.id && p.validate(proposal.Indexes, proposal.Root) {
			if len(proposal.Indexes) == 0 {
				p.log.Infof("[Proposal %d] - Received empty pool. Skipping.", p.id)
			}

			p.log.Infof("[Proposal %d] - Pool root: %s", p.id, proposal.Root)
			p.log.Infof("[Proposal %d] - Indexes: %v", p.id, proposal.Indexes)

			p.result <- &session.Proposal{
				Indexes: proposal.Indexes,
				Root:    proposal.Root,
			}
		}
	}

	return nil
}

func (p *ProposalController) validate(indexes []string, root string) bool {
	contents, err := p.getContents(context.Background(), indexes)
	if err != nil {
		p.log.WithError(err).Error("error preparing contents")
		return false
	}

	return hexutil.Encode(merkle.NewTree(crypto.Keccak256, contents...).Root()) == root
}

func (p *ProposalController) Run(ctx context.Context) {
	p.wg.Add(1)
	go p.run(ctx)
}

func (p *ProposalController) run(ctx context.Context) {
	if p.proposer.PubKey != p.secret.ECDSAPubKeyStr() {
		p.log.Infof("[Proposal %d] - Proposer is another party", p.id)
		return
	}

	ids, root, err := p.getNewPool(ctx)
	if err != nil {
		p.log.WithError(err).Error("[Proposal %d] - Error preparing pool to propose", p.id)
		return
	}

	p.log.Infof("[Proposal %d] - Pool root %s", p.id, root)
	p.log.Infof("[Proposal %d] - Indexes: %v", p.id, ids)

	p.result <- &session.Proposal{
		Indexes: ids,
		Root:    root,
	}

	details, err := cosmostypes.NewAnyWithValue(&types.ProposalRequest{Session: p.id, Indexes: ids, Root: root})
	if err != nil {
		p.log.WithError(err).Error("error parsing details")
		return
	}

	p.connector.SubmitAll(ctx, &types.MsgSubmitRequest{
		Type:    types.RequestType_Proposal,
		Details: details,
	})

	p.log.Infof("[Proposal %d] - Controller finished", p.id)
	p.wg.Done()
}

func (p *ProposalController) getNewPool(ctx context.Context) ([]string, string, error) {
	ids, err := p.pool.GetNext(MaxPoolSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "error preparing pool")
	}

	if len(ids) == 0 {
		p.log.Infof("[Proposal %d] - Empty pool. Skipping.", p.id)
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
		case rarimo.OpType_CHANGE_KEY:
			content, err := p.getChangeContent(ctx, resp.Operation)
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

func (p *ProposalController) getChangeContent(_ context.Context, op rarimo.Operation) (merkle.Content, error) {
	change, err := pkg.GetChangeKey(op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	content, err := pkg.GetChangeKeyContent(change)
	return content, errors.Wrap(err, "error creating content")
}

func (p *ProposalController) WaitFinish() {
	p.wg.Wait()
}
