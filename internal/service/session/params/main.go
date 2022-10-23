package params

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/session"
)

// Storage handles tss parameters from core and called up the source for the parameters in all components.
type Storage struct {
	params   *rarimo.Params
	toNotify map[string]session.ParamsNotifier

	log *logan.Entry
}

func NewParamsStorage(cfg config.Config) (*Storage, error) {
	resp, err := rarimo.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return &Storage{
		params: &resp.Params,
		log:    cfg.Log(),
	}, nil
}

// session.IParamsStorage implementation

var _ session.IParamsStorage = &Storage{}

func (p *Storage) GetParams() *rarimo.Params {
	return p.params
}

func (p *Storage) UpdateParams(params *rarimo.Params) {
	p.params = params
	go p.notifyAll(params)
}

func (p *Storage) SubscribeToParams(name string, f session.ParamsNotifier) {
	p.toNotify[name] = f
	go p.notify(p.params, name, f)
}

func (p *Storage) notifyAll(params *rarimo.Params) {
	for name, f := range p.toNotify {
		p.notify(params, name, f)
	}
}

func (p *Storage) notify(params *rarimo.Params, name string, f session.ParamsNotifier) {
	if err := f(params); err != nil {
		p.log.WithError(err).Errorf("error notifying for new params %s", name)
	}
}
