package session

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

type ParamsNotifier func(params *rarimo.Params) error

// ParamsSaver handles tss parameters from core and called up the source for the parameters in all components.
type ParamsSaver struct {
	params   *rarimo.Params
	toNotify map[string]ParamsNotifier

	log *logan.Entry
}

func NewParamsSaver(cfg config.Config) (*ParamsSaver, error) {
	resp, err := rarimo.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return &ParamsSaver{
		params: &resp.Params,
		log:    cfg.Log(),
	}, nil
}

func (p *ParamsSaver) GetParams() *rarimo.Params {
	return p.params
}

func (p *ParamsSaver) UpdateParams(params *rarimo.Params) {
	p.params = params
	go p.notifyAll(params)
}

func (p *ParamsSaver) SubscribeToParams(name string, f ParamsNotifier) {
	p.toNotify[name] = f
	go p.notify(p.params, name, f)
}

func (p *ParamsSaver) notifyAll(params *rarimo.Params) {
	for name, f := range p.toNotify {
		p.notify(params, name, f)
	}
}

func (p *ParamsSaver) notify(params *rarimo.Params, name string, f ParamsNotifier) {
	if err := f(params); err != nil {
		p.log.WithError(err).Errorf("error notifying for new params %s", name)
	}
}
