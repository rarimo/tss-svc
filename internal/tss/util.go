package tss

import (
	"github.com/bnb-chain/tss-lib/tss"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
)

type waitingMessage struct {
	sender      *rarimo.Party
	isBroadcast bool
	details     []byte
}

func partiesByAccountMapping(parties []*rarimo.Party) map[string]*rarimo.Party {
	pmap := make(map[string]*rarimo.Party)
	for _, p := range parties {
		pmap[p.Account] = p
	}
	return pmap
}

func logPartyStatus(log *logan.Entry, party tss.Party, self string) {
	list := party.WaitingFor()
	monikers := make([]string, 0, len(list))
	for _, p := range list {
		if p.Moniker == self {
			monikers = append(monikers, "SELF")
			continue
		}

		monikers = append(monikers, p.Moniker)
	}
	log.Infof("Waiting for messages from: %v", monikers)
	log.Infof("Party status: %s", party.String())
}
