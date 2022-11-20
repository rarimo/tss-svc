package core

import rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"

func Equal(p1 *rarimo.Party, p2 *rarimo.Party) bool {
	return p1.Address == p2.Account
}
