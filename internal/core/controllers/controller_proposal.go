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
	merkle "gitlab.com/rarimo/go-merkle"
	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/pool"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const MaxPoolSize = 32

// iProposalController defines custom logic for every proposal controller.
type iProposalController interface {
	accept(details *cosmostypes.Any, st types.SessionType) bool
	shareProposal(ctx context.Context)
	updateSessionData()
}

// ProposalController is responsible for proposing and collecting proposals from proposer.
// Proposer will execute logic of defining the next session type and suggest data to suggest in session.
type ProposalController struct {
	iProposalController
	wg *sync.WaitGroup

	data *LocalSessionData

	auth    *core.RequestAuthorizer
	factory *ControllerFactory
	log     *logan.Entry
}

// Implements IController interface
var _ IController = &ProposalController{}

// Receive accepts proposal from other parties. It will check that proposal was submitted from the selected session proposer.
// After it will execute the `iProposalController.accept` logic.
func (p *ProposalController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := p.auth.Auth(request)
	if err != nil {
		return err
	}

	if !Equal(sender, &p.data.Proposer) {
		p.data.Offenders[sender.Account] = struct{}{}
		return ErrSenderIsNotProposer
	}

	if request.Type != types.RequestType_Proposal {
		return ErrInvalidRequestType
	}

	p.log.Infof("Received proposal request from %s for session type=%s", sender.Account, request.SessionType.String())
	if accepted := p.accept(request.Details, request.SessionType); !accepted {
		p.data.Offenders[sender.Account] = struct{}{}
	}

	return nil
}

// Run launches the sharing proposal logic in corresponding session in case of self party is a session proposer.
func (p *ProposalController) Run(ctx context.Context) {
	p.log.Infof("Starting: %s", p.Type().String())
	p.wg.Add(1)
	go p.run(ctx)
}

// WaitFor waits until controller finishes its logic. Context cancel should be called before.
func (p *ProposalController) WaitFor() {
	p.wg.Wait()
}

// Next will return acceptance controller if proposal sharing or receiving was successful,
// otherwise, it will return finish controller.
// WaitFor should be called before.
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
		p.log.Infof("Finishing: %s", p.Type().String())
		p.updateSessionData()
		p.wg.Done()
	}()

	p.log.Debugf("Session %s %d proposer: %v", p.data.SessionType.String(), p.data.SessionId, p.data.Proposer)

	if p.data.Proposer.Account != p.data.Secret.AccountAddress() {
		p.log.Debug("Proposer is another party. No actions required")
		return
	}

	p.shareProposal(ctx)
	<-ctx.Done()
}

// defaultProposalController represents custom logic for types.SessionType_DefaultSession
type defaultProposalController struct {
	mu        sync.Mutex
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	core      *connectors.CoreConnector
	client    *grpc.ClientConn
	pg        *pg.Storage
	log       *logan.Entry
}

// Implements iProposalController interface
var _ iProposalController = &defaultProposalController{}

// accept will check the received proposal to sign the set of operations by validating suggested indexes.
// If the current parties is not active (set contains inactive parties or party was removed) the flow will be reverted.
func (d *defaultProposalController) accept(details *cosmostypes.Any, st types.SessionType) bool {
	if st != types.SessionType_DefaultSession || !d.data.Set.IsActive {
		return false
	}

	data := new(types.DefaultSessionProposalData)
	if err := proto.Unmarshal(details.Value, data); err != nil {
		d.log.WithError(err).Error("Error unmarshalling request")
		return false
	}

	d.log.Infof("Proposal request details: indexes=%v root=%s", data.Indexes, data.Root)
	if len(data.Indexes) == 0 {
		return false
	}

	ops, err := GetOperations(d.client, data.Indexes...)
	if err != nil {
		return false
	}

	contents, err := GetContents(d.client, ops...)
	if err != nil {
		return false
	}

	if hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()) == data.Root {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.log.Infof("Proposal data is correct. Proposal accepted.")
		d.data.Processing = true
		d.data.Root = data.Root
		d.data.Indexes = data.Indexes
		return true
	}

	return false
}

// shareProposal selects the operation indexes from the pool, constructs the proposal and share it between parties.
// If the current parties is not active (set contains inactive parties or party was removed) the flow will be reverted.
func (d *defaultProposalController) shareProposal(ctx context.Context) {
	// Unable to perform signing if set is inactive, need to perform reshare first
	if !d.data.Set.IsActive {
		return
	}

	d.log.Debugf("Making sign proposal")
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

	go d.broadcast.SubmitAllWithReport(ctx, d.core, &types.MsgSubmitRequest{
		Id:          d.data.SessionId,
		Type:        types.RequestType_Proposal,
		IsBroadcast: true,
		Details:     details,
	})

	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.Root = root
	d.data.Indexes = ids
	d.data.Processing = true
}

// updateSessionData updates the database entry according to the controller result.
func (d *defaultProposalController) updateSessionData() {
	d.mu.Lock()
	defer d.mu.Unlock()

	session, err := d.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(int64(d.data.SessionId), false)
	if err != nil {
		d.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		d.log.Error("Session entry is not initialized")
		return
	}

	session.Parties = partyAccounts(d.data.Set.Parties)
	session.Proposer = sql.NullString{
		String: d.data.Proposer.Account,
		Valid:  true,
	}
	session.Indexes = d.data.Indexes
	session.Root = sql.NullString{
		String: d.data.Root,
		Valid:  d.data.Root != "",
	}

	if err = d.pg.DefaultSessionDatumQ().Update(session); err != nil {
		d.log.WithError(err).Error("Error updating session entry")
	}
}

func (d *defaultProposalController) getNewPool() ([]string, string, error) {
	ids, err := pool.GetPool().GetNext(MaxPoolSize)
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

// reshareProposalController represents custom logic for types.SessionType_ReshareSession
type reshareProposalController struct {
	mu        sync.Mutex
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	core      *connectors.CoreConnector
	log       *logan.Entry
	pg        *pg.Storage
}

// Implements iProposalController interface
var _ iProposalController = &reshareProposalController{}

// accept will check received proposal to reshare keys corresponding to the party local data.
func (r *reshareProposalController) accept(details *cosmostypes.Any, st types.SessionType) bool {
	if st != types.SessionType_ReshareSession || r.data.Set.IsActive {
		return false
	}

	data := new(types.ReshareSessionProposalData)
	if err := proto.Unmarshal(details.Value, data); err != nil {
		r.log.WithError(err).Error("Error unmarshalling request")
		return false
	}

	r.log.Infof("Proposal request details: Set = %v", data.Set)
	if checkSet(data.Set, r.data.Set) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.log.Infof("Proposal data is correct. Proposal accepted.")
		r.data.Processing = true
		return true
	}

	return false
}

// shareProposal constructs the reshare proposal based on local party data and shares it between the parties.
func (r *reshareProposalController) shareProposal(ctx context.Context) {
	// Unable to perform signing if set is active or public key does not exist.
	if r.data.Set.IsActive || r.data.Set.GlobalPubKey == "" {
		return
	}

	r.log.Debugf("Making reshare proposal")
	set := getSet(r.data.Set)
	data := &types.ReshareSessionProposalData{Set: set}

	r.log.Infof("Performed set for updating to: %v", set)

	details, err := cosmostypes.NewAnyWithValue(data)
	if err != nil {
		r.log.WithError(err).Error("Error parsing data")
		return
	}

	go r.broadcast.SubmitAllWithReport(ctx, r.core, &types.MsgSubmitRequest{
		Id:          r.data.SessionId,
		Type:        types.RequestType_Proposal,
		IsBroadcast: true,
		Details:     details,
	})

	r.mu.Lock()
	defer r.mu.Unlock()
	r.data.Processing = true
}

// updateSessionData updates the database entry according to the controller result.
func (r *reshareProposalController) updateSessionData() {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, err := r.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(int64(r.data.SessionId), false)
	if err != nil {
		r.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		r.log.Error("Session entry is not initialized")
		return
	}

	session.Parties = partyAccounts(r.data.Set.Parties)
	session.Proposer = sql.NullString{
		String: r.data.Proposer.Account,
		Valid:  true,
	}
	session.OldKey = sql.NullString{
		String: r.data.Set.GlobalPubKey,
		Valid:  true,
	}

	if err = r.pg.ReshareSessionDatumQ().Update(session); err != nil {
		r.log.WithError(err).Error("Error updating session entry")
	}
}
