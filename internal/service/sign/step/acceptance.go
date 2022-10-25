package step

import (
	"context"
	"sync"

	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	mu   sync.Mutex
	root string
	tssP *rarimo.Params

	result chan *session.Acceptance

	acceptances []string
	index       map[string]struct{}

	log *logan.Entry
}

func (a *AcceptanceController) ReceiveAcceptance(sender string, request types.MsgSubmitRequest) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.index[sender]; !ok && request.Type == types.RequestType_Proposal {
		acceptance := new(types.AcceptanceRequest)

		if err := proto.Unmarshal(request.Details.Value, acceptance); err != nil {
			a.log.WithError(err).Error("error unmarshalling details")
			return
		}

		if acceptance.Root == a.root {
			a.index[sender] = struct{}{}
			a.acceptances = append(a.acceptances, sender)
		}
	}
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
