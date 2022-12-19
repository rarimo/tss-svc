package tss

import rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"

func partiesByAccountMapping(parties []*rarimo.Party) map[string]*rarimo.Party {
	pmap := make(map[string]*rarimo.Party)
	for _, p := range parties {
		pmap[p.Account] = p
	}
	return pmap
}
