package step

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	root string

	result chan *session.Signature

	log *logan.Entry
}

func (s *SignatureController) ReceiveSign(sender string, request types.MsgSubmitRequest) {
	if request.Type == types.RequestType_Sign {
		sign := new(types.SignRequest)

		if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
			s.log.WithError(err).Error("error unmarshalling details")
			return
		}

		// TODO
	}
}

func (s *SignatureController) Run(ctx context.Context) {
	go s.run(ctx)
}

func (s *SignatureController) run(ctx context.Context) {
	// TODo
}
