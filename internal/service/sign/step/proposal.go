package step

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	merkle "gitlab.com/rarify-protocol/go-merkle"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

type ProposalController struct {
	id       uint64
	proposer *rarimo.Party

	result chan *session.Proposal
	pool   *pool.Pool
	rarimo *grpc.ClientConn
	params *local.Storage
	log    *logan.Entry
}

func NewProposalController(
	id uint64,
	params *local.Storage,
	proposer *rarimo.Party,
	result chan *session.Proposal,
	pool *pool.Pool,
	rarimo *grpc.ClientConn,
	log *logan.Entry,
) *ProposalController {
	return &ProposalController{
		id:       id,
		params:   params,
		proposer: proposer,
		result:   result,
		pool:     pool,
		rarimo:   rarimo,
		log:      log,
	}
}

func (p *ProposalController) ReceiveProposal(sender *rarimo.Party, request types.MsgSubmitRequest) error {
	if request.Type == types.RequestType_Proposal && sender.PubKey == p.proposer.PubKey {
		proposal := new(types.ProposalRequest)

		if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
			return err
		}

		if proposal.Session == p.id {
			p.result <- &session.Proposal{
				Indexes: proposal.Indexes,
				Root:    proposal.Root,
			}
		}
	}

	return nil
}

func (p *ProposalController) Run(ctx context.Context) {
	go p.run(ctx)
}

func (p *ProposalController) run(ctx context.Context) {
	// TODO check who is proposer
	// if its me then:
	ids, root, err := p.getNewPool(ctx)
	if err != nil {
		p.log.WithError(err).Error("error preparing pool to propose")
		return
	}

	p.result <- &session.Proposal{
		Indexes: ids,
		Root:    root,
	}

	// TODO broadcast
}

func (p *ProposalController) getNewPool(ctx context.Context) ([]string, string, error) {
	ids, err := p.pool.GetNext(sign.MaxPoolSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "error preparing pool")
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
			return nil, sign.ErrUnsupportedContent
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
