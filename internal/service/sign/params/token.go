package params

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

type TokenParamsNotifier func(params *token.Params) error

// TokenStorage handles token manager parameters from core
// and called up the source for the parameters in all components.
type TokenStorage struct {
	params   *token.Params
	toNotify map[string]TokenParamsNotifier

	log *logan.Entry
}

func NewTokenParamsStorage(cfg config.Config) (*TokenStorage, error) {
	resp, err := token.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return &TokenStorage{
		params: &resp.Params,
		log:    cfg.Log(),
	}, nil
}

func (t *TokenStorage) GetParams() *token.Params {
	return t.params
}

func (t *TokenStorage) UpdateParams(params *token.Params) {
	t.params = params
	go t.notifyAll(params)
}

func (t *TokenStorage) SubscribeToParams(name string, f TokenParamsNotifier) {
	t.toNotify[name] = f
	go t.notify(t.params, name, f)
}

func (t *TokenStorage) notifyAll(params *token.Params) {
	for name, f := range t.toNotify {
		t.notify(params, name, f)
	}
}

func (t *TokenStorage) notify(params *token.Params, name string, f TokenParamsNotifier) {
	if err := f(params); err != nil {
		t.log.WithError(err).Errorf("error notifying for new params %s", name)
	}
}
