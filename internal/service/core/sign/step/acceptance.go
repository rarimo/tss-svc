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
	root string

	acceptances []string
	index       map[string]struct{}

	result chan *session.Acceptance

	connector *connectors.BroadcastConnector
	params    *local.Params
	log       *logan.Entry
}

func NewAcceptanceController(
	root string,
	result chan *session.Acceptance,
	connector *connectors.BroadcastConnector,
	params *local.Params,
	log *logan.Entry,
) *AcceptanceController {
	return &AcceptanceController{
		root:        root,
		params:      params,
		result:      result,
		acceptances: make([]string, 0, params.N()),
		index:       make(map[string]struct{}),
		connector:   connector,
		log:         log,
	}
}

var _ IController = &AcceptanceController{}

func (a *AcceptanceController) Receive(sender rarimo.Party, request types.MsgSubmitRequest) error {
	if _, ok := a.index[sender.PubKey]; !ok && request.Type == types.RequestType_Proposal {
		acceptance := new(types.AcceptanceRequest)

		if err := proto.Unmarshal(request.Details.Value, acceptance); err != nil {
			return err
		}

		if acceptance.Root == a.root {
			a.index[sender.PubKey] = struct{}{}
			a.acceptances = append(a.acceptances, sender.PubKey)
		}
	}

	return nil
}

func (a *AcceptanceController) Run(ctx context.Context) {
	go a.run(ctx)
}

func (a *AcceptanceController) run(ctx context.Context) {
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

	if len(a.result) >= a.params.T() {
		a.result <- &session.Acceptance{
			Accepted: a.acceptances,
		}
	}
}
