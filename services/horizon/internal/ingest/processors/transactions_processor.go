package processors

import (
	"context"

	"github.com/pownieh/stellar_go/ingest"
	"github.com/pownieh/stellar_go/services/horizon/internal/db2/history"
	"github.com/pownieh/stellar_go/support/db"
	"github.com/pownieh/stellar_go/support/errors"
	"github.com/pownieh/stellar_go/xdr"
)

type TransactionProcessor struct {
	batch history.TransactionBatchInsertBuilder
}

func NewTransactionFilteredTmpProcessor(batch history.TransactionBatchInsertBuilder) *TransactionProcessor {
	return &TransactionProcessor{
		batch: batch,
	}
}

func NewTransactionProcessor(batch history.TransactionBatchInsertBuilder) *TransactionProcessor {
	return &TransactionProcessor{
		batch: batch,
	}
}

func (p *TransactionProcessor) ProcessTransaction(lcm xdr.LedgerCloseMeta, transaction ingest.LedgerTransaction) error {
	if err := p.batch.Add(transaction, lcm.LedgerSequence()); err != nil {
		return errors.Wrap(err, "Error batch inserting transaction rows")
	}

	return nil
}

func (p *TransactionProcessor) Flush(ctx context.Context, session db.SessionInterface) error {
	return p.batch.Exec(ctx, session)
}
