package controllers

import (
	"context"
	"database/sql"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	merkle "github.com/rarimo/go-merkle"
	"github.com/rarimo/tss-svc/internal/connectors"
	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

const MaxPoolSize = 32

// iProposalController defines custom logic for every proposal controller.
type iProposalController interface {
	accept(ctx core.Context, details *anypb.Any, st types.SessionType) bool
	shareProposal(ctx core.Context)
	updateSessionData(ctx core.Context)
}

// ProposalController is responsible for proposing and collecting proposals from proposer.
// Proposer will execute logic of defining the next session type and suggest data to suggest in session.
type ProposalController struct {
	iProposalController
	wg   *sync.WaitGroup
	data *LocalSessionData
	auth *core.RequestAuthorizer
}

// Implements IController interface
var _ IController = &ProposalController{}

// Receive accepts proposal from other parties. It will check that proposal was submitted from the selected session proposer.
// After it will execute the `iProposalController.accept` logic.
func (p *ProposalController) Receive(c context.Context, request *types.MsgSubmitRequest) error {
	ctx := core.WrapCtx(c)

	sender, err := p.auth.Auth(request)
	if err != nil {
		return err
	}

	if !Equal(sender, &p.data.Proposer) {
		p.data.Offenders[sender.Account] = struct{}{}
		return ErrSenderIsNotProposer
	}

	if request.Data.Type != types.RequestType_Proposal {
		return ErrInvalidRequestType
	}

	ctx.Log().Infof("Received proposal request from %s for session type=%s", sender.Account, request.Data.SessionType.String())
	if accepted := p.accept(ctx, request.Data.Details, request.Data.SessionType); !accepted {
		p.data.Offenders[sender.Account] = struct{}{}
	}

	return nil
}

// Run launches the sharing proposal logic in corresponding session in case of self party is a session proposer.
func (p *ProposalController) Run(c context.Context) {
	ctx := core.WrapCtx(c)
	ctx.Log().Infof("Starting: %s", p.Type().String())
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
		return p.data.GetAcceptanceController()
	}

	return p.data.GetFinishController()
}

func (p *ProposalController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_PROPOSAL
}

func (p *ProposalController) run(ctx core.Context) {
	defer func() {
		ctx.Log().Infof("Finishing: %s", p.Type().String())
		p.updateSessionData(ctx)
		p.wg.Done()
	}()

	ctx.Log().Debugf("Session %s %d proposer: %v", p.data.SessionType.String(), p.data.SessionId, p.data.Proposer)

	if p.data.Proposer.Account != ctx.SecretStorage().GetTssSecret().AccountAddress() {
		ctx.Log().Debug("Proposer is another party. No actions required")
		return
	}

	p.shareProposal(ctx)
	<-ctx.Context().Done()
}

// defaultProposalController represents custom logic for types.SessionType_DefaultSession
type defaultProposalController struct {
	mu        sync.Mutex
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
}

// Implements iProposalController interface
var _ iProposalController = &defaultProposalController{}

// accept will check the received proposal to sign the set of operations by validating suggested indexes.
// If the current parties is not active (set contains inactive parties or party was removed) the flow will be reverted.
func (d *defaultProposalController) accept(ctx core.Context, details *anypb.Any, st types.SessionType) bool {
	if st != types.SessionType_DefaultSession || !d.data.Set.IsActive {
		return false
	}

	data := new(types.DefaultSessionProposalData)

	if err := details.UnmarshalTo(data); err != nil {
		ctx.Log().WithError(err).Error("Error unmarshalling request")
		return false
	}

	ctx.Log().Infof("Proposal request details: indexes=%v root=%s", data.Indexes, data.Root)
	if len(data.Indexes) == 0 {
		return false
	}

	ops, err := GetOperations(ctx.Client(), data.Indexes...)
	if err != nil {
		return false
	}

	contents, err := GetContents(ctx.Client(), ops...)
	if err != nil {
		return false
	}

	if hexutil.Encode(merkle.NewTree(eth.Keccak256, contents...).Root()) == data.Root {
		d.mu.Lock()
		defer d.mu.Unlock()
		ctx.Log().Infof("Proposal data is correct. Proposal accepted.")
		d.data.Processing = true
		d.data.Root = data.Root
		d.data.Indexes = data.Indexes
		return true
	}

	return false
}

// shareProposal selects the operation indexes from the pool, constructs the proposal and share it between parties.
// If the current parties is not active (set contains inactive parties or party was removed) the flow will be reverted.
func (d *defaultProposalController) shareProposal(ctx core.Context) {
	// Unable to perform signing if set is inactive, need to perform reshare first
	if !d.data.Set.IsActive {
		return
	}

	ctx.Log().Debugf("Making sign proposal")
	ids, root, err := d.getNewPool(ctx)
	if err != nil {
		ctx.Log().WithError(err).Error("Error preparing pool to propose")
		return
	}

	if len(ids) == 0 {
		ctx.Log().Info("Empty pool. Skipping.")
		return
	}

	ctx.Log().Infof("Performed pool to share: %v", ids)

	details, err := anypb.New(&types.DefaultSessionProposalData{Indexes: ids, Root: root})
	if err != nil {
		ctx.Log().WithError(err).Error("Error parsing data")
		return
	}

	go d.broadcast.SubmitAllWithReport(ctx.Context(), ctx.Core(), &types.MsgSubmitRequest{
		Data: &types.RequestData{
			SessionType: types.SessionType_DefaultSession,
			Type:        types.RequestType_Proposal,
			Id:          d.data.SessionId,
			IsBroadcast: true,
			Details:     details,
		},
	})

	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.Root = root
	d.data.Indexes = ids
	d.data.Processing = true
}

// updateSessionData updates the database entry according to the controller result.
func (d *defaultProposalController) updateSessionData(ctx core.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()

	session, err := ctx.PG().DefaultSessionDatumQ().DefaultSessionDatumByID(int64(d.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
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

	if err = ctx.PG().DefaultSessionDatumQ().Update(session); err != nil {
		ctx.Log().WithError(err).Error("Error updating session entry")
	}
}

func (d *defaultProposalController) getNewPool(ctx core.Context) ([]string, string, error) {
	ids, err := ctx.Pool().GetNext(MaxPoolSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "error preparing pool")
	}

	if len(ids) == 0 {
		return []string{}, "", nil
	}

	ops, err := GetOperations(ctx.Client(), ids...)
	if err != nil {
		return nil, "", err
	}

	contents, err := GetContents(ctx.Client(), ops...)
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
}

// Implements iProposalController interface
var _ iProposalController = &reshareProposalController{}

// accept will check received proposal to reshare keys corresponding to the party local data.
func (r *reshareProposalController) accept(ctx core.Context, details *anypb.Any, st types.SessionType) bool {
	if st != types.SessionType_ReshareSession || r.data.Set.IsActive {
		return false
	}

	data := new(types.ReshareSessionProposalData)
	if err := details.UnmarshalTo(data); err != nil {
		ctx.Log().WithError(err).Error("Error unmarshalling request")
		return false
	}

	ctx.Log().Infof("Proposal request details: Set = %v", data.Set)
	if checkSet(data.Set, r.data.Set) {
		r.mu.Lock()
		defer r.mu.Unlock()
		ctx.Log().Infof("Proposal data is correct. Proposal accepted.")
		r.data.Processing = true
		return true
	}

	return false
}

// shareProposal constructs reshare proposal based on local party data and shares it between the parties.
func (r *reshareProposalController) shareProposal(ctx core.Context) {
	// Unable to perform signing if set is active or public key does not exist.
	if r.data.Set.IsActive || r.data.Set.GlobalPubKey == "" {
		return
	}

	ctx.Log().Debugf("Making reshare proposal")
	set := getSet(r.data.Set)
	data := &types.ReshareSessionProposalData{Set: set}

	ctx.Log().Infof("Performed set for updating to: %v", set)

	details, err := anypb.New(data)
	if err != nil {
		ctx.Log().WithError(err).Error("Error parsing data")
		return
	}

	go r.broadcast.SubmitAllWithReport(ctx.Context(), ctx.Core(), &types.MsgSubmitRequest{
		Data: &types.RequestData{
			SessionType: types.SessionType_ReshareSession,
			Type:        types.RequestType_Proposal,
			Id:          r.data.SessionId,
			IsBroadcast: true,
			Details:     details,
		},
	})

	r.mu.Lock()
	defer r.mu.Unlock()
	r.data.Processing = true
}

// updateSessionData updates the database entry according to the controller result.
func (r *reshareProposalController) updateSessionData(ctx core.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, err := ctx.PG().ReshareSessionDatumQ().ReshareSessionDatumByID(int64(r.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
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

	if err = ctx.PG().ReshareSessionDatumQ().Update(session); err != nil {
		ctx.Log().WithError(err).Error("Error updating session entry")
	}
}
