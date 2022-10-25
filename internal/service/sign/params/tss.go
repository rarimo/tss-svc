package params

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

type TSSParamsNotifier func(params *rarimo.Params) error

// TSSStorage handles tss parameters from core
// and called up the source for the parameters in all components.
type TSSStorage struct {
	params   *rarimo.Params
	toNotify map[string]TSSParamsNotifier

	log *logan.Entry
}

func NewTSSParamsStorage(cfg config.Config) (*TSSStorage, error) {
	resp, err := rarimo.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return &TSSStorage{
		params: &resp.Params,
		log:    cfg.Log(),
	}, nil
}

func (t *TSSStorage) GetParams() *rarimo.Params {
	return t.params
}

func (t *TSSStorage) UpdateParams(params *rarimo.Params) {
	t.params = params
	go t.notifyAll(params)
}

func (t *TSSStorage) SubscribeToParams(name string, f TSSParamsNotifier) {
	t.toNotify[name] = f
	go t.notify(t.params, name, f)
}

func (t *TSSStorage) notifyAll(params *rarimo.Params) {
	for name, f := range t.toNotify {
		t.notify(params, name, f)
	}
}

func (t *TSSStorage) notify(params *rarimo.Params, name string, f TSSParamsNotifier) {
	if err := f(params); err != nil {
		t.log.WithError(err).Errorf("error notifying for new params %s", name)
	}
}
