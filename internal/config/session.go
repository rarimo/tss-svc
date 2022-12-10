package config

import (
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
)

type SessionInfo struct {
	StartBlock     uint64 `fig:"start_block"`
	StartSessionId uint64 `fig:"start_session_id"`
}

func (c *config) Session() *SessionInfo {
	return c.session.Do(func() interface{} {
		info := &SessionInfo{}
		if err := figure.Out(info).From(kv.MustGetStringMap(c.getter, "session")).Please(); err != nil {
			panic(err)
		}
		return info
	}).(*SessionInfo)
}
