package processors

import (
	"context"

	"github.com/pownieh/stellar_go/ingest"
	"github.com/pownieh/stellar_go/services/horizon/internal/db2/history"
	"github.com/pownieh/stellar_go/support/collections/set"
	"github.com/pownieh/stellar_go/support/db"
	"github.com/pownieh/stellar_go/support/errors"
	"github.com/pownieh/stellar_go/toid"
	"github.com/pownieh/stellar_go/xdr"
)

type LiquidityPoolsTransactionProcessor struct {
	lpLoader *history.LiquidityPoolLoader
	txBatch  history.TransactionLiquidityPoolBatchInsertBuilder
	opBatch  history.OperationLiquidityPoolBatchInsertBuilder
}

func NewLiquidityPoolsTransactionProcessor(
	lpLoader *history.LiquidityPoolLoader,
	txBatch history.TransactionLiquidityPoolBatchInsertBuilder,
	opBatch history.OperationLiquidityPoolBatchInsertBuilder,
) *LiquidityPoolsTransactionProcessor {
	return &LiquidityPoolsTransactionProcessor{
		lpLoader: lpLoader,
		txBatch:  txBatch,
		opBatch:  opBatch,
	}
}

func (p *LiquidityPoolsTransactionProcessor) ProcessTransaction(lcm xdr.LedgerCloseMeta, transaction ingest.LedgerTransaction) error {
	err := p.addTransactionLiquidityPools(lcm.LedgerSequence(), transaction)
	if err != nil {
		return err
	}

	err = p.addOperationLiquidityPools(lcm.LedgerSequence(), transaction)
	if err != nil {
		return err
	}

	return nil
}

func (p *LiquidityPoolsTransactionProcessor) addTransactionLiquidityPools(sequence uint32, transaction ingest.LedgerTransaction) error {
	transactionID := toid.New(int32(sequence), int32(transaction.Index), 0).ToInt64()
	lps, err := liquidityPoolsForTransaction(transaction)
	if err != nil {
		return errors.Wrap(err, "Could not determine liquidity pools for transaction")
	}

	for _, lp := range dedupeStrings(lps) {
		if err = p.txBatch.Add(transactionID, p.lpLoader.GetFuture(lp)); err != nil {
			return err
		}
	}

	return nil
}

func liquidityPoolsForTransaction(transaction ingest.LedgerTransaction) ([]string, error) {
	changes, err := transaction.GetChanges()
	if err != nil {
		return nil, err
	}
	lps, err := liquidityPoolsForChanges(changes)
	if err != nil {
		return nil, errors.Wrapf(err, "reading transaction %v liquidity pools", transaction.Index)
	}
	return lps, nil
}

func dedupeStrings(in []string) []string {
	set := set.Set[string]{}
	for _, id := range in {
		set.Add(id)
	}

	out := make([]string, 0, len(in))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func liquidityPoolsForChanges(
	changes []ingest.Change,
) ([]string, error) {
	var lps []string

	for _, c := range changes {
		if c.Type != xdr.LedgerEntryTypeLiquidityPool {
			continue
		}

		if c.Pre == nil && c.Post == nil {
			return nil, errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
		}

		if c.Pre != nil {
			poolID := c.Pre.Data.MustLiquidityPool().LiquidityPoolId
			lps = append(lps, PoolIDToString(poolID))
		}
		if c.Post != nil {
			poolID := c.Post.Data.MustLiquidityPool().LiquidityPoolId
			lps = append(lps, PoolIDToString(poolID))
		}
	}

	return lps, nil
}

func (p *LiquidityPoolsTransactionProcessor) addOperationLiquidityPools(sequence uint32, transaction ingest.LedgerTransaction) error {
	for opi, op := range transaction.Envelope.Operations() {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		changes, err := transaction.GetOperationChanges(uint32(opi))
		if err != nil {
			return err
		}
		lps, err := liquidityPoolsForChanges(changes)
		if err != nil {
			return errors.Wrapf(err, "reading operation %v liquidity pools", operation.ID())
		}
		for _, lp := range dedupeStrings(lps) {
			if err := p.opBatch.Add(operation.ID(), p.lpLoader.GetFuture(lp)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *LiquidityPoolsTransactionProcessor) Flush(ctx context.Context, session db.SessionInterface) error {
	if err := p.txBatch.Exec(ctx, session); err != nil {
		return errors.Wrap(err, "Could not flush transaction liquidity pools to db")
	}
	if err := p.opBatch.Exec(ctx, session); err != nil {
		return errors.Wrap(err, "Could not flush operation liquidity pools to db")
	}

	return nil
}
