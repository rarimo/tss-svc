package core

import (
	"time"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
)

const (
	minViolationCount = 15
	maxViolationCount = 20
	forgivenessDelay  = 10 * time.Minute
)

type counter struct {
	cnt  uint
	last time.Time
}

type RatCounter struct {
	counter map[string]counter
	params  *local.Params
}

func NewRatCounter(cfg config.Config) *RatCounter {
	return &RatCounter{
		counter: make(map[string]counter),
		params:  local.NewParams(cfg),
	}
}

func (r *RatCounter) RegisterAcceptances(accounts map[string]bool) {
	parties := make(map[string]bool)
	for _, p := range r.params.Parties() {
		parties[p.Account] = true
	}

	for a := range accounts {
		delete(parties, a)
	}

	for p := range parties {
		counter := r.counter[p]
		counter.cnt++
		counter.last = time.Now().UTC()
		r.counter[p] = counter
	}
}

func (r *RatCounter) GetRats() []string {
	rats := make([]string, 0, r.params.N())
	for acc, counter := range r.counter {
		if counter.cnt >= maxViolationCount {
			rats = append(rats, acc)
		}
	}

	return rats
}

func (r *RatCounter) IsRat(account string) bool {
	return r.counter[account].cnt >= minViolationCount
}

func (r *RatCounter) Update() {
	for acc, c := range r.counter {
		if c.cnt > 0 && c.last.Add(forgivenessDelay).Before(time.Now().UTC()) {
			r.counter[acc] = counter{
				cnt:  c.cnt - 1,
				last: time.Now().UTC(),
			}
		}
	}
}

func (r *RatCounter) PossibleChange(change *rarimo.ChangeParties) bool {
	parties := make(map[string]struct{})
	for _, p := range r.params.Parties() {
		parties[p.Account] = struct{}{}
	}

	for _, a := range change.NewSet {
		delete(parties, a.Account)
	}

	for p := range parties {
		if r.counter[p].cnt < minViolationCount {
			return false
		}
	}

	return true
}
