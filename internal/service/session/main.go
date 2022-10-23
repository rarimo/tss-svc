package session

import rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"

type IPool interface {
	Add(id string) error
	GetNext(n uint) ([]string, error)
}

type BlockNotifier func(height uint64) error

type ITimer interface {
	NewBlock(height uint64)
	CurrentBlock() uint64
	SubscribeToBlocks(name string, f BlockNotifier)
}

type ParamsNotifier func(params *rarimo.Params) error

type IParamsStorage interface {
	UpdateParams(params *rarimo.Params)
	GetParams() *rarimo.Params
	SubscribeToParams(name string, f ParamsNotifier)
}
