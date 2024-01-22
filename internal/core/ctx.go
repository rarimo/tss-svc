package core

import (
	"net"

	"github.com/rarimo/tss-svc/internal/config"
	"github.com/rarimo/tss-svc/internal/connectors"
	"github.com/rarimo/tss-svc/internal/data/pg"
	"github.com/rarimo/tss-svc/internal/pool"
	"github.com/rarimo/tss-svc/internal/secret"
	"github.com/rarimo/tss-svc/internal/timer"
	"github.com/rarimo/tss-svc/pkg/types"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type (
	RegistryKey uint
	ContextKey  uint

	registry struct {
		registry map[RegistryKey]any
	}

	Context struct {
		ctx context.Context
	}
)

const ContextTypeKey = "context_key"

const (
	GlobalContextKey         ContextKey = iota
	DefaultSessionContextKey ContextKey = iota
	KeygenSessionContextKey  ContextKey = iota
	ReshareSessionContextKey ContextKey = iota
)

const (
	PGKey RegistryKey = iota
	SecretKey
	LogKey
	ClientKey
	CoreKey
	PoolKey
	TimerKey
	TendermintKey
	ListenerKey
	SwaggerKey
)

var (
	registries = make(map[ContextKey]*registry)

	ContextKeyBySessionType = map[types.SessionType]ContextKey{
		types.SessionType_DefaultSession: DefaultSessionContextKey,
		types.SessionType_ReshareSession: ReshareSessionContextKey,
		types.SessionType_KeygenSession:  KeygenSessionContextKey,
	}
)

func SetInRegistry(ctxKey ContextKey, key RegistryKey, value any) {
	rg := registries[ctxKey]
	if rg == nil {
		rg = &registry{
			registry: make(map[RegistryKey]any),
		}
		registries[ctxKey] = rg
	}

	rg.registry[key] = value
}

func Initialize(cfg config.Config) {
	db := pg.New(cfg.DB())
	SetInRegistry(GlobalContextKey, PGKey, db)
	SetInRegistry(DefaultSessionContextKey, PGKey, db)
	SetInRegistry(ReshareSessionContextKey, PGKey, db)
	SetInRegistry(KeygenSessionContextKey, PGKey, db)

	secret := secret.NewVaultStorage(cfg)
	SetInRegistry(GlobalContextKey, SecretKey, secret)
	SetInRegistry(DefaultSessionContextKey, SecretKey, secret)
	SetInRegistry(ReshareSessionContextKey, SecretKey, secret)
	SetInRegistry(KeygenSessionContextKey, SecretKey, secret)

	SetInRegistry(GlobalContextKey, ClientKey, cfg.Cosmos())
	SetInRegistry(DefaultSessionContextKey, ClientKey, cfg.Cosmos())
	SetInRegistry(ReshareSessionContextKey, ClientKey, cfg.Cosmos())
	SetInRegistry(KeygenSessionContextKey, ClientKey, cfg.Cosmos())

	core := connectors.NewCoreConnector(cfg.Cosmos(), secret.GetTssSecret(), cfg.Log(), cfg.ChainParams())
	SetInRegistry(GlobalContextKey, CoreKey, core)
	SetInRegistry(DefaultSessionContextKey, CoreKey, core)
	SetInRegistry(ReshareSessionContextKey, CoreKey, core)
	SetInRegistry(KeygenSessionContextKey, CoreKey, core)

	pool := pool.NewPool(cfg)
	SetInRegistry(GlobalContextKey, PoolKey, pool)
	SetInRegistry(DefaultSessionContextKey, PoolKey, pool)

	timer := timer.NewTimer(cfg.Tendermint(), cfg.Log())
	SetInRegistry(GlobalContextKey, TimerKey, timer)

	SetInRegistry(GlobalContextKey, TendermintKey, cfg.Tendermint())

	SetInRegistry(GlobalContextKey, LogKey, cfg.Log())

	SetInRegistry(GlobalContextKey, ListenerKey, cfg.Listener())

	SetInRegistry(GlobalContextKey, SwaggerKey, cfg.Swagger())
	SetInRegistry(DefaultSessionContextKey, SwaggerKey, cfg.Swagger())
	SetInRegistry(ReshareSessionContextKey, SwaggerKey, cfg.Swagger())
	SetInRegistry(KeygenSessionContextKey, SwaggerKey, cfg.Swagger())
}

func WrapCtx(ctx context.Context) Context {
	st := ctx.Value(ContextTypeKey).(ContextKey)
	rg := registries[st]
	if rg == nil {
		panic("registry not found")
	}

	for k, v := range rg.registry {
		ctx = context.WithValue(ctx, k, v)
	}

	return Context{ctx: ctx}
}

func GetSessionCtx(ctx context.Context, sessionType types.SessionType) context.Context {
	return addCtxTypeKey(ctx, ContextKeyBySessionType[sessionType])
}

func DefaultSessionContext(sessionType types.SessionType) Context {
	return WrapCtx(GetSessionCtx(context.TODO(), sessionType))
}

func DefaultGlobalContext(ctx context.Context) Context {
	return WrapCtx(addCtxTypeKey(ctx, GlobalContextKey))
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) PG() *pg.Storage {
	return c.ctx.Value(PGKey).(*pg.Storage)
}

func (c *Context) SecretStorage() secret.Storage {
	return c.ctx.Value(SecretKey).(secret.Storage)
}

func (c *Context) Log() *logan.Entry {
	return c.ctx.Value(LogKey).(*logan.Entry)
}

func (c *Context) Client() *grpc.ClientConn {
	return c.ctx.Value(ClientKey).(*grpc.ClientConn)
}

func (c *Context) Core() *connectors.CoreConnector {
	return c.ctx.Value(CoreKey).(*connectors.CoreConnector)
}

func (c *Context) Pool() *pool.Pool {
	return c.ctx.Value(PoolKey).(*pool.Pool)
}

func (c *Context) Timer() *timer.Timer {
	return c.ctx.Value(TimerKey).(*timer.Timer)
}

func (c *Context) Tendermint() *http.HTTP {
	return c.ctx.Value(TendermintKey).(*http.HTTP)
}

func (c *Context) Listener() net.Listener {
	return c.ctx.Value(ListenerKey).(net.Listener)
}

func (c *Context) Swagger() *config.SwaggerInfo {
	return c.ctx.Value(SwaggerKey).(*config.SwaggerInfo)
}

func addCtxTypeKey(ctx context.Context, key ContextKey) context.Context {
	return context.WithValue(ctx, ContextTypeKey, key)
}
