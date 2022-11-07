package step

import (
	"context"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	mu   sync.Mutex
	wg   *sync.WaitGroup
	id   uint64
	root string

	acceptances []string
	index       map[string]struct{}

	result chan *session.Acceptance

	connector *connectors.BroadcastConnector
	params    *local.Params
	secret    *local.Secret
	log       *logan.Entry
}

func NewAcceptanceController(
	id uint64,
	root string,
	result chan *session.Acceptance,
	connector *connectors.BroadcastConnector,
	params *local.Params,
	secret *local.Secret,
	log *logan.Entry,
) *AcceptanceController {
	return &AcceptanceController{
		wg:          &sync.WaitGroup{},
		id:          id,
		root:        root,
		params:      params,
		secret:      secret,
		result:      result,
		acceptances: make([]string, 0, params.N()),
		index:       make(map[string]struct{}),
		connector:   connector,
		log:         log,
	}
}

var _ IController = &AcceptanceController{}

func (a *AcceptanceController) Run(ctx context.Context) {
	a.wg.Add(1)
	go a.run(ctx)
}

func (a *AcceptanceController) ReceiveFromSender(sender rarimo.Party, request *types.MsgSubmitRequest) {
	if _, ok := a.index[sender.Account]; !ok && request.Type == types.RequestType_Proposal {
		acceptance := new(types.AcceptanceRequest)
		if err := proto.Unmarshal(request.Details.Value, acceptance); err != nil {
			a.log.WithError(err).Error("error unmarshalling request")
		}

		if acceptance.Root == a.root {
			a.log.Infof("[Acceptance %d] - Received acceptance from %s for root %s ---", a.id, sender.Account, a.root)
			a.index[sender.Account] = struct{}{}
			a.acceptances = append(a.acceptances, sender.Account)
		}
	}
}

func (a *AcceptanceController) run(ctx context.Context) {
	a.acceptances = append(a.acceptances, a.secret.AccountAddressStr())

	details, err := cosmostypes.NewAnyWithValue(&types.AcceptanceRequest{Root: a.root})
	if err != nil {
		a.log.WithError(err).Error("error parsing details")
		return
	}

	a.connector.SubmitAll(ctx, &types.MsgSubmitRequest{
		Type:    types.RequestType_Acceptance,
		Details: details,
	})

	<-ctx.Done()
	a.log.Infof("[Acceptance %d] - Acceptances: %v", a.id, a.acceptances)

	if len(a.acceptances) >= a.params.T() {
		a.log.Infof("[Acceptance %d] - Reached required amount of acceptances", a.id)

		a.result <- &session.Acceptance{
			Accepted: a.acceptances,
		}
	}
	a.log.Infof("[Acceptance %d] - Controller finished", a.id)
	a.wg.Done()
}

func (a *AcceptanceController) WaitFinish() {
	a.wg.Wait()
}
