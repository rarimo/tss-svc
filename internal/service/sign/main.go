package sign

import (
	goerr "errors"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors/party"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/params"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/step"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/timer"
	"google.golang.org/grpc"
)

const (
	MaxPoolSize        = 32
	StepProposingIndex = 0
	StepAcceptingIndex = 1
	StepSigningIndex   = 2
)

var (
	ErrUnsupportedContent = goerr.New("unsupported content")
)

type Service struct {
	mu sync.Mutex

	pool         *pool.Pool
	timer        *timer.Timer
	con          *party.SubmitConnector
	tssStorage   *params.TSSStorage
	tokenStorage *params.TokenStorage

	step    *step.Step
	session *session.Session

	log     *logan.Entry
	rarimo  *grpc.ClientConn
	storage *pg.Storage
}

// NewBlock receives new blocks from timer
func (s *Service) NewBlock(height uint64) error {
	return nil
}

/*func (p *ProposalController) nextProposer(signature string, nextSessionId uint64) *rarimo.Party {
	sigBytes := hexutil.MustDecode(signature)
	stepBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(stepBytes, nextSessionId)
	hash := crypto.Keccak256(sigBytes, stepBytes)
	return p.tssP.Parties[int(hash[len(hash)-1])%len(p.tssP.Parties)]
}
*/
