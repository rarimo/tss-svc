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

// ProposalController is responsible for proposing and collecting proposals from proposer.
// Proposer will execute logic of defining the next session type and suggest data to process in session.
type ProposalController struct {
	IProposalController
	wg *sync.WaitGroup

	data *LocalSessionData

	auth    *core.RequestAuthorizer
	factory *ControllerFactory
	log     *logan.Entry
}

// Implements IController interface
var _ IController = &ProposalController{}

func (p *ProposalController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := p.auth.Auth(request)
	if err != nil {
		return err
	}

	if !core.Equal(sender, &p.data.Proposer) {
		return ErrSenderIsNotProposer
	}

	if request.Type != types.RequestType_Proposal {
		return ErrInvalidRequestType
	}

	proposal := new(types.ProposalRequest)
	if err := proto.Unmarshal(request.Details.Value, proposal); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	p.log.Infof("Received proposal request from %s for session type=%s", sender.Account, proposal.Type.String())
	p.accept(proposal.Details, proposal.Type)
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
	if p.data.Processing {
		return p.factory.GetAcceptanceController()
	}

	return p.factory.GetFinishController()
}

func (p *ProposalController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_PROPOSAL
}

func (p *ProposalController) run(ctx context.Context) {
	defer func() {
		p.log.Infof("%s finished", p.Type().String())
		p.updateSessionData()
		p.wg.Done()
	}()

	if p.data.Proposer.Account != p.data.Secret.AccountAddress() {
		p.log.Info("Proposer is another party")
		return
	}

	p.shareProposal(ctx)
	<-ctx.Done()
}

// IProposalController defines custom logic for every proposal controller.
type IProposalController interface {
	accept(details *cosmostypes.Any, st types.SessionType)
	shareProposal(ctx context.Context)
	updateSessionData()
}

// DefaultProposalController represents custom logic for types.SessionType_DefaultSession
type DefaultProposalController struct {
	mu        sync.Mutex
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	client    *grpc.ClientConn
	pool      *pool.Pool
	pg        *pg.Storage
	log       *logan.Entry
}

// Implements IProposalController interface
var _ IProposalController = &DefaultProposalController{}

func (d *DefaultProposalController) accept(details *cosmostypes.Any, st types.SessionType) {
	if st != types.SessionType_DefaultSession {
		return
	}

	data := new(types.DefaultSessionProposalData)
	if err := proto.Unmarshal(details.Value, data); err != nil {
		d.log.WithError(err).Error("error unmarshalling request")
		return
	}

	d.log.Infof("Proposal request details: indexes=%v root=%s", data.Indexes, data.Root)
	if len(data.Indexes) == 0 {
		return
	}

	ops, err := GetOperations(d.client, data.Indexes...)
	if err != nil {
		return
	}

	contents, err := GetContents(d.client, ops...)
	if err != nil {
		return
	}

	if hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()) == data.Root {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.log.Infof("Proposal data is correct")
		d.data.SessionType = types.SessionType_DefaultSession
		d.data.Processing = true
		d.data.Root = data.Root
		d.data.Indexes = data.Indexes
	}
}

func (d *DefaultProposalController) shareProposal(ctx context.Context) {
	d.log.Infof("Making sign proposal")
	ids, root, err := d.getNewPool()
	if err != nil {
		d.log.WithError(err).Error("Error preparing pool to propose")
		return
	}

	if len(ids) == 0 {
		d.log.Info("Empty pool. Skipping.")
		return
	}

	d.log.Infof("Performed pool to share: %v", ids)

	details, err := cosmostypes.NewAnyWithValue(&types.DefaultSessionProposalData{Indexes: ids, Root: root})
	if err != nil {
		d.log.WithError(err).Error("Error parsing data")
		return
	}

	details, err = cosmostypes.NewAnyWithValue(&types.ProposalRequest{Type: types.SessionType_DefaultSession, Details: details})
	if err != nil {
		d.log.WithError(err).Error("Error parsing details")
		return
	}

	d.broadcast.SubmitAll(ctx, &types.MsgSubmitRequest{
		Id:          d.data.SessionId,
		Type:        types.RequestType_Proposal,
		IsBroadcast: true,
		Details:     details,
	})

	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.SessionType = types.SessionType_DefaultSession
	d.data.Root = root
	d.data.Indexes = ids
	d.data.Processing = true
}

func (d *DefaultProposalController) updateSessionData() {
	d.mu.Lock()
	defer d.mu.Unlock()

	session, err := d.pg.SessionQ().SessionByID(int64(d.data.SessionId), false)
	if err != nil {
		d.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		d.log.Error("session entry is not initialized")
		return
	}

	session.SessionType = sql.NullInt64{
		Int64: int64(types.SessionType_DefaultSession),
		Valid: true,
	}

	err = d.pg.DefaultSessionDatumQ().Insert(&data.DefaultSessionDatum{
		ID:      session.ID,
		Parties: partyAccounts(d.data.Set.Parties),
		Proposer: sql.NullString{
			String: d.data.Proposer.Account,
			Valid:  true,
		},
		Indexes: d.data.Indexes,
		Root: sql.NullString{
			String: d.data.Root,
			Valid:  d.data.Root != "",
		},
	})

	if err != nil {
		d.log.WithError(err).Error("error creating session data entry")
		return
	}

	session.DataID = sql.NullInt64{
		Int64: session.ID,
		Valid: true,
	}

	if err = d.pg.SessionQ().Update(session); err != nil {
		d.log.WithError(err).Error("error updating session entry")
	}
}

func (d *DefaultProposalController) getNewPool() ([]string, string, error) {
	ids, err := d.pool.GetNext(MaxPoolSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "error preparing pool")
	}

	if len(ids) == 0 {
		return []string{}, "", nil
	}

	ops, err := GetOperations(d.client, ids...)
	if err != nil {
		return nil, "", err
	}

	contents, err := GetContents(d.client, ops...)
	if err != nil {
		return nil, "", err
	}

	return ids, hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()), nil
}

// ReshareProposalController represents custom logic for types.SessionType_ReshareSession
type ReshareProposalController struct {
	mu        sync.Mutex
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	log       *logan.Entry
	pg        *pg.Storage
}

// Implements IProposalController interface
var _ IProposalController = &ReshareProposalController{}

func (r *ReshareProposalController) accept(details *cosmostypes.Any, st types.SessionType) {
	if st != types.SessionType_ReshareSession {
		return
	}

	data := new(types.ReshareSessionProposalData)
	if err := proto.Unmarshal(details.Value, data); err != nil {
		r.log.WithError(err).Error("error unmarshalling request")
		return
	}

	r.log.Infof("Proposal request details: new=%v", data.New)
	if checkSet(data.New, r.data.Set) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.log.Infof("Proposal data is correct")
		r.data.SessionType = types.SessionType_ReshareSession
		r.data.Processing = true
	}
}

func (r *ReshareProposalController) shareProposal(ctx context.Context) {
	r.log.Infof("Making reshare proposal")
	data := &types.ReshareSessionProposalData{
		// TODO
		Old:          getSet(nil),
		New:          getSet(nil),
		OldPublicKey: "",
	}

	details, err := cosmostypes.NewAnyWithValue(data)
	if err != nil {
		r.log.WithError(err).Error("Error parsing data")
		return
	}

	details, err = cosmostypes.NewAnyWithValue(&types.ProposalRequest{Type: types.SessionType_ReshareSession, Details: details})
	if err != nil {
		r.log.WithError(err).Error("Error parsing details")
		return
	}

	r.broadcast.SubmitAll(ctx, &types.MsgSubmitRequest{
		Id:          r.data.SessionId,
		Type:        types.RequestType_Proposal,
		IsBroadcast: true,
		Details:     details,
	})

	r.mu.Lock()
	defer r.mu.Unlock()
	r.data.SessionType = types.SessionType_ReshareSession
	r.data.Processing = true
}

func (r *ReshareProposalController) updateSessionData() {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, err := r.pg.SessionQ().SessionByID(int64(r.data.SessionId), false)
	if err != nil {
		r.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		r.log.Error("session entry is not initialized")
		return
	}

	session.SessionType = sql.NullInt64{
		Int64: int64(types.SessionType_ReshareSession),
		Valid: true,
	}

	err = r.pg.ReshareSessionDatumQ().Insert(&data.ReshareSessionDatum{
		ID:      session.ID,
		Parties: partyAccounts(r.data.Set.Parties),
		Proposer: sql.NullString{
			String: r.data.Proposer.Account,
			Valid:  true,
		},
		OldKey: sql.NullString{
			String: r.data.Set.GlobalPubKey,
			Valid:  true,
		},
	})

	if err != nil {
		r.log.WithError(err).Error("error creating session data entry")
		return
	}

	session.DataID = sql.NullInt64{
		Int64: session.ID,
		Valid: true,
	}

	if err = r.pg.SessionQ().Update(session); err != nil {
		r.log.WithError(err).Error("error updating session entry")
	}
}
