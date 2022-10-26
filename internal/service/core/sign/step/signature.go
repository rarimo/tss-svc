package step

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	root string

	result chan *session.Signature
	params *local.Params
	log    *logan.Entry
}

func NewSignatureController(
	root string,
	params *local.Params,
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

var _ IController = &SignatureController{}

func (s *SignatureController) Receive(sender rarimo.Party, request types.MsgSubmitRequest) error {
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
