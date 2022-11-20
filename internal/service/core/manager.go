package core

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var manager *Manager

type Manager struct {
	cancel  context.CancelFunc
	current IController
}

func NewManager(c IController) *Manager {
	if manager == nil {
		manager = &Manager{
			current: c,
		}
		manager.run()
	} else if c != nil {
		panic("cannot re-initialize manager")
	}

	return manager
}

func (m *Manager) NewBlock(height uint64) error {
	if m.current.End() >= height {
		m.cancel()
		m.current.WaitFor()
		m.current = m.current.Next()
		m.run()
	}
	return nil
}

func (m *Manager) run() {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.TODO())
	m.current.Run(ctx)
}

func (m *Manager) Receive(req *types.MsgSubmitRequest) error {
	return m.current.Receive(req)
}
