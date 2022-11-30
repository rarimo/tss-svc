package controllers

import (
	"context"
	"database/sql"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	merkle "gitlab.com/rarify-protocol/go-merkle"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const MaxPoolSize = 32

type ProposalController struct {
	mu sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	log       *logan.Entry

	client  *grpc.ClientConn
	pool    *pool.Pool
	pg      *pg.Storage
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

	switch proposal.Type {
	case types.SessionType_DefaultSession:
		data := new(types.DefaultSessionProposalData)
		if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
			return errors.Wrap(err, "error unmarshalling details")
		}

		if p.validateDefaultProposal(data) {
			func() {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.data.SessionType = types.SessionType_DefaultSession
				p.data.Processing = true
				p.data.Root = data.Root
				p.data.Indexes = data.Indexes
			}()

		}
	case types.SessionType_ReshareSession:
		data := new(types.ReshareSessionProposalData)
		if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
			return errors.Wrap(err, "error unmarshalling details")
		}

		if p.validateReshareProposal(data) {
			func() {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.data.SessionType = types.SessionType_ReshareSession
				p.data.Processing = true
			}()
		}
	}

	return nil
}

func (p *ProposalController) Run(ctx context.Context) {
	p.log.Infof("Starting %s", p.Type().String())
	p.wg.Add(1)
	go p.run(ctx)
}

func (p *ProposalController) WaitFor() {
	p.wg.Wait()
}

func (p *ProposalController) Next() IController {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.data.SessionType {
	case types.SessionType_DefaultSession:
		// We're definitely working on the old set
		p.data.New = p.data.Old
		if p.data.Processing {
			return p.factory.GetAcceptanceController()
		}
	case types.SessionType_ReshareSession:
		if p.data.Processing {
			return p.factory.GetAcceptanceController()
		}
	}

	return p.factory.GetFinishController()
}

func (p *ProposalController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_PROPOSAL
}

func (p *ProposalController) validateDefaultProposal(data *types.DefaultSessionProposalData) bool {
	if len(data.Indexes) == 0 {
		return false
	}

	ops, err := GetOperations(p.client, data.Indexes...)
	if err != nil {
		return false
	}

	contents, err := GetContents(p.client, ops...)
	if err != nil {
		return false
	}

	return hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()) == data.Root
}

func (p *ProposalController) validateReshareProposal(data *types.ReshareSessionProposalData) bool {
	if p.data.Old.Equals(p.data.New) {
		return false
	}

	return checkSet(data.Old, p.data.Old) && checkSet(data.New, p.data.New)
}

func (p *ProposalController) run(ctx context.Context) {
	defer func() {
		p.log.Infof("%s finished", p.Type().String())
		p.updateSessionData()
		p.wg.Done()
	}()

	if p.data.Proposer.Account != p.data.New.LocalAccountAddress {
		p.log.Info("Proposer is another party")
		return
	}

	if p.data.Old.Equals(p.data.New) {
		p.makeSignProposal(ctx)
		return
	}

	p.makeReshareProposal(ctx)
}

func (p *ProposalController) makeSignProposal(ctx context.Context) {
	ids, root, err := p.getNewPool()
	if err != nil {
		p.log.WithError(err).Error("Error preparing pool to propose")
		return
	}

	if len(ids) == 0 {
		p.log.Info("Empty pool. Skipping.")
		return
	}

	details, err := cosmostypes.NewAnyWithValue(&types.DefaultSessionProposalData{Indexes: ids, Root: root})
	if err != nil {
		p.log.WithError(err).Error("Error parsing data")
		return
	}

	details, err = cosmostypes.NewAnyWithValue(&types.ProposalRequest{Type: types.SessionType_DefaultSession, Details: details})
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

	p.mu.Lock()
	defer p.mu.Unlock()
	p.data.SessionType = types.SessionType_DefaultSession
	p.data.Root = root
	p.data.Indexes = ids
	p.data.Processing = true
}

func (p *ProposalController) makeReshareProposal(ctx context.Context) {
	data := &types.ReshareSessionProposalData{
		Old:          getSet(p.data.Old),
		New:          getSet(p.data.New),
		OldPublicKey: p.data.Old.GlobalPubKey,
	}

	details, err := cosmostypes.NewAnyWithValue(data)
	if err != nil {
		p.log.WithError(err).Error("Error parsing data")
		return
	}

	details, err = cosmostypes.NewAnyWithValue(&types.ProposalRequest{Type: types.SessionType_ReshareSession, Details: details})
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

	p.mu.Lock()
	defer p.mu.Unlock()
	p.data.SessionType = types.SessionType_ReshareSession
	p.data.Processing = true
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

	return ids, hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()), nil
}

func (p *ProposalController) updateSessionData() {
	session, err := p.pg.SessionQ().SessionByID(int64(p.data.SessionId), false)
	if err != nil {
		p.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		p.log.Error("session entry is not initialized")
		return
	}

	switch p.data.SessionType {
	case types.SessionType_DefaultSession:
		session.SessionType = sql.NullInt64{
			Int64: int64(types.SessionType_DefaultSession),
			Valid: true,
		}

		err = p.pg.DefaultSessionDatumQ().Insert(&data.DefaultSessionDatum{
			ID:      session.ID,
			Parties: partyAccounts(p.data.New.Parties),
			Proposer: sql.NullString{
				String: p.data.Proposer.Account,
				Valid:  true,
			},
			Indexes: p.data.Indexes,
			Root: sql.NullString{
				String: p.data.Root,
				Valid:  p.data.Root != "",
			},
		})
	case types.SessionType_ReshareSession:
		session.SessionType = sql.NullInt64{
			Int64: int64(types.SessionType_ReshareSession),
			Valid: true,
		}

		err = p.pg.ReshareSessionDatumQ().Insert(&data.ReshareSessionDatum{
			ID:      session.ID,
			Parties: partyAccounts(p.data.New.Parties),
			Proposer: sql.NullString{
				String: p.data.Proposer.Account,
				Valid:  true,
			},
			OldKey: sql.NullString{
				String: p.data.Old.GlobalPubKey,
				Valid:  true,
			},
		})
	}

	if err != nil {
		p.log.WithError(err).Error("error creating session data entry")
		return
	}

	session.DataID = sql.NullInt64{
		Int64: session.ID,
		Valid: true,
	}

	if err = p.pg.SessionQ().Update(session); err != nil {
		p.log.WithError(err).Error("error updating session entry")
	}
}
