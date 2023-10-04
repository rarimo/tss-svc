package controllers

import (
	"context"
	"encoding/binary"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	merkle "github.com/rarimo/go-merkle"
	"github.com/rarimo/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	token "github.com/rarimo/rarimo-core/x/tokenmanager/types"
	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/internal/pool"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"google.golang.org/grpc"
)

// GetProposer generates deterministic proposer based on linear congruential generator with seed from getHash(signature, sessionId)
func GetProposer(parties []*rarimo.Party, sig string, sessionId uint64) rarimo.Party {
	rnd := newRnd(new(big.Int).SetBytes(getHash(sig, sessionId)))
	index := getIndex(rnd.next(), len(parties))
	return *parties[index]
}

// GetSignersSet generates deterministic signers set based on received acceptances
// and linear congruential generator with seed from getHash(signature, sessionId)
func GetSignersSet(acceptances map[string]struct{}, t int, sig string, sessionId uint64) map[string]struct{} {
	accepted := acceptancesToArr(acceptances)
	rnd := newRnd(new(big.Int).SetBytes(getHash(sig, sessionId)))

	sort.Strings(accepted)
	result := make(map[string]struct{})

	for i := 0; i <= t; i++ {
		index := getIndex(rnd.next(), len(accepted))
		result[accepted[index]] = struct{}{}

		// in case of it was last element we need to exclude it from set just by deleting last slice element
		if index == len(accepted)-1 {
			accepted = accepted[:index]
			continue
		}

		// otherwise we should concatenate 0...index-1 and index+1..end
		accepted = append(accepted[:index], accepted[index+1:]...)
	}

	return result
}

func getHash(lastSignature string, sessionId uint64) []byte {
	sigBytes := hexutil.MustDecode(lastSignature)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, sessionId)
	return eth.Keccak256(sigBytes, idBytes)
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
		if op.Status != rarimo.OpStatus_APPROVED {
			return nil, pool.ErrOpShouldBeApproved
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
			return nil, ErrUnsupportedContent
		case rarimo.OpType_FEE_TOKEN_MANAGEMENT:
			content, err := GetFeeManagementContent(client, op)
			if err != nil {
				return nil, err
			}

			if content != nil {
				contents = append(contents, content)
			}
		case rarimo.OpType_CONTRACT_UPGRADE:
			content, err := GetContractUpgradeContent(client, op)
			if err != nil {
				return nil, err
			}

			if content != nil {
				contents = append(contents, content)
			}
		case rarimo.OpType_IDENTITY_DEFAULT_TRANSFER:
			content, err := GetIdentityDefaultTransferContent(op)
			if err != nil {
				return nil, err
			}

			if content != nil {
				contents = append(contents, content)
			}
		case rarimo.OpType_IDENTITY_AGGREGATED_TRANSFER:
			content, err := GetIdentityAggregatedTransferContent(op)
			if err != nil {
				return nil, err
			}

			if content != nil {
				contents = append(contents, content)
			}
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

	collectionDataResp, err := token.NewQueryClient(client).CollectionData(context.TODO(), &token.QueryGetCollectionDataRequest{Chain: transfer.To.Chain, Address: transfer.To.Address})
	if err != nil {
		return nil, errors.Wrap(err, "error getting collection data entry")
	}

	collectionResp, err := token.NewQueryClient(client).Collection(context.TODO(), &token.QueryGetCollectionRequest{Index: collectionDataResp.Data.Collection})
	if err != nil {
		return nil, errors.Wrap(err, "error getting collection data entry")
	}

	onChainItemResp, err := token.NewQueryClient(client).OnChainItem(context.TODO(), &token.QueryGetOnChainItemRequest{Chain: transfer.To.Chain, Address: transfer.To.Address, TokenID: transfer.To.TokenID})
	if err != nil {
		return nil, errors.Wrap(err, "error getting on chain item entry")
	}

	itemResp, err := token.NewQueryClient(client).Item(context.TODO(), &token.QueryGetItemRequest{Index: onChainItemResp.Item.Item})
	if err != nil {
		return nil, errors.Wrap(err, "error getting item entry")
	}

	networkResp, err := token.NewQueryClient(client).NetworkParams(context.TODO(), &token.QueryNetworkParamsRequest{Name: transfer.To.Chain})
	if err != nil {
		return nil, errors.Wrap(err, "error getting network param entry")
	}

	bridgeparams := networkResp.Params.GetBridgeParams()
	if err != nil {
		return nil, errors.New("bridge params not found")
	}

	content, err := pkg.GetTransferContent(collectionResp.Collection, collectionDataResp.Data, itemResp.Item, bridgeparams, transfer)
	return content, errors.Wrap(err, "error creating content")
}

func GetFeeManagementContent(client *grpc.ClientConn, op *rarimo.Operation) (merkle.Content, error) {
	manage, err := pkg.GetFeeTokenManagement(*op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	networkResp, err := token.NewQueryClient(client).NetworkParams(context.TODO(), &token.QueryNetworkParamsRequest{Name: manage.Chain})
	if err != nil {
		return nil, errors.Wrap(err, "error getting network param entry")
	}

	feeparams := networkResp.Params.GetFeeParams()
	if err != nil {
		return nil, errors.New("bridge params not found")
	}

	content, err := pkg.GetFeeTokenManagementContent(feeparams, manage)
	return content, errors.Wrap(err, "error creating content")
}

func GetContractUpgradeContent(client *grpc.ClientConn, op *rarimo.Operation) (merkle.Content, error) {
	upgrade, err := pkg.GetContractUpgrade(*op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	networkResp, err := token.NewQueryClient(client).NetworkParams(context.TODO(), &token.QueryNetworkParamsRequest{Name: upgrade.Chain})
	if err != nil {
		return nil, errors.Wrap(err, "error getting network param entry")
	}

	content, err := pkg.GetContractUpgradeContent(networkResp.Params, upgrade)
	return content, errors.Wrap(err, "error creating content")
}

func GetIdentityDefaultTransferContent(op *rarimo.Operation) (merkle.Content, error) {
	transfer, err := pkg.GetIdentityDefaultTransfer(*op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	content, err := pkg.GetIdentityDefaultTransferContent(transfer)
	return content, errors.Wrap(err, "error creating content")
}

func GetIdentityAggregatedTransferContent(op *rarimo.Operation) (merkle.Content, error) {
	transfer, err := pkg.GetIdentityAggregatedTransfer(*op)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing operation details")
	}

	content, err := pkg.GetIdentityAggregatedTransferContent(transfer)
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

func getSignersList(all map[string]struct{}, parties []*rarimo.Party) []*rarimo.Party {
	accepted := make([]*rarimo.Party, 0, len(parties))

	for _, p := range parties {
		if _, ok := all[p.Account]; ok {
			accepted = append(accepted, p)
		}
	}

	return accepted
}

func filterAcceptances(all map[string]struct{}, parties []*rarimo.Party) map[string]struct{} {
	filtered := make(map[string]struct{})

	for _, p := range parties {
		if _, ok := all[p.Account]; ok {
			filtered[p.Account] = struct{}{}
		}
	}

	return filtered
}

func contains(list []*rarimo.Party, account string) bool {
	for _, p := range list {
		if p.Account == account {
			return true
		}
	}

	return false
}
