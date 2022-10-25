package step

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	root string

	result chan *session.Signature
	params *local.Storage
	log    *logan.Entry
}

func NewSignatureController(
	root string,
	params *local.Storage,
	result chan *session.Signature,
	log *logan.Entry,
) *SignatureController {
	return &SignatureController{
		root:   root,
		params: params,
		result: result,
		log:    log,
	}
}

func (s *SignatureController) ReceiveSign(sender *rarimo.Party, request types.MsgSubmitRequest) error {
	if request.Type == types.RequestType_Sign {
		sign := new(types.SignRequest)

		if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
			return err
		}

		if sign.Root == s.root {
			// TODO
		}
	}
	return nil
}

func (s *SignatureController) Run(ctx context.Context) {
	go s.run(ctx)
}

func (s *SignatureController) run(ctx context.Context) {
	// TODo
}
