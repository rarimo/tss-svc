package controllers

import (
	"context"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3/errors"
	merkle "gitlab.com/rarimo/go-merkle"
	"gitlab.com/rarimo/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/pool"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

func GetProposer(parties []*rarimo.Party, sig string, sessionId uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(sig)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, sessionId)
	hash := eth.Keccak256(sigBytes, idBytes)
	return *parties[int(hash[len(hash)-1])%len(parties)]
}

func Equal(p1 *rarimo.Party, p2 *rarimo.Party) bool {
	return p1.Account == p2.Account
}

func GetOperations(client *grpc.ClientConn, ids ...string) ([]*rarimo.Operation, error) {
	operations := make([]*rarimo.Operation, 0, len(ids))

	for _, id := range ids {
		resp, err := rarimo.NewQueryClient(client).Operation(context.TODO(), &rarimo.QueryGetOperationRequest{Index: id})
		if err != nil {
			return nil, errors.Wrap(err, "error fetching operation")
		}

		operations = append(operations, &resp.Operation)
	}

	return operations, nil
}

func GetContents(client *grpc.ClientConn, operations ...*rarimo.Operation) ([]merkle.Content, error) {
	contents := make([]merkle.Content, 0, len(operations))

	for _, op := range operations {
		if op.Signed {
			return nil, pool.ErrOpAlreadySigned
		}

		switch op.OperationType {
		case rarimo.OpType_TRANSFER:
			content, err := GetTransferContent(client, op)
			if err != nil {
				return nil, err
			}

			if content != nil {
				contents = append(contents, content)
			}

		case rarimo.OpType_CHANGE_PARTIES:
			// Currently not supported here
		default:
			return nil, ErrUnsupportedContent
		}
	}

	return contents, nil
}

func GetTransferContent(client *grpc.ClientConn, op *rarimo.Operation) (merkle.Content, error) {
	transfer, err := pkg.GetTransfer(*op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	infoResp, err := token.NewQueryClient(client).Info(context.TODO(), &token.QueryGetInfoRequest{Index: transfer.TokenIndex})
	if err != nil {
		return nil, errors.Wrap(err, "error getting token info entry")
	}

	itemResp, err := token.NewQueryClient(client).Item(context.TODO(), &token.QueryGetItemRequest{
		TokenAddress: infoResp.Info.Chains[transfer.ToChain].TokenAddress,
		TokenId:      infoResp.Info.Chains[transfer.ToChain].TokenId,
		Chain:        transfer.ToChain,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error getting token item entry")
	}

	params, err := token.NewQueryClient(client).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "error getting params")
	}

	content, err := pkg.GetTransferContent(&itemResp.Item, params.Params.Networks[transfer.ToChain], transfer)
	return content, errors.Wrap(err, "error creating content")
}

func checkSet(proposal *types.Set, input *core.InputSet) bool {
	if len(proposal.Parties) != len(input.Parties) || int(proposal.N) != input.N || int(proposal.T) != input.T {
		return false
	}

	for i := range proposal.Parties {
		if proposal.Parties[i] != input.Parties[i].Account {
			return false
		}
	}
	return true
}

func getSet(input *core.InputSet) *types.Set {
	res := &types.Set{
		Parties: make([]string, 0, input.N),
		N:       uint32(input.N),
		T:       uint32(input.T),
	}

	for _, p := range input.Parties {
		res.Parties = append(res.Parties, p.Account)
	}
	return res
}

func partyAccounts(parties []*rarimo.Party) []string {
	res := make([]string, 0, len(parties))
	for _, p := range parties {
		res = append(res, p.Account)
	}
	return res
}

func acceptancesToArr(acc map[string]struct{}) []string {
	res := make([]string, 0, len(acc))
	for p, _ := range acc {
		res = append(res, p)
	}
	return res
}

func getPartiesAcceptances(all map[string]struct{}, parties []*rarimo.Party) []*rarimo.Party {
	accepted := make([]*rarimo.Party, 0, len(parties))

	for _, p := range parties {
		if _, ok := all[p.Account]; ok {
			accepted = append(accepted, p)
		}
	}

	return accepted
}

func contains(list []*rarimo.Party, account string) bool {
	for _, p := range list {
		if p.Account == account {
			return true
		}
	}

	return false
}
