package step

import (
	"context"
	"sync"

	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	mu   sync.Mutex
	root string

	acceptances []string
	index       map[string]struct{}

	result chan *session.Acceptance
	params *local.Storage
	log    *logan.Entry
}

func NewAcceptanceController(
	root string,
	result chan *session.Acceptance,
	params *local.Storage,
	log *logan.Entry,
) *AcceptanceController {
	return &AcceptanceController{
		root:        root,
		params:      params,
		result:      result,
		acceptances: make([]string, 0, params.N()),
		index:       make(map[string]struct{}),
		log:         log,
	}
}

func (a *AcceptanceController) ReceiveAcceptance(sender *rarimo.Party, request types.MsgSubmitRequest) error {
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
	_ = types.AcceptanceRequest{Root: a.root}

	// TODO broadcast acceptance

	<-ctx.Done()
	// TODO check minimum t acceptances
	a.result <- &session.Acceptance{
		Accepted: a.acceptances,
	}
}
