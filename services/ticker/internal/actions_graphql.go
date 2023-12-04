package ticker

import (
	"github.com/pownieh/stellar_go/services/ticker/internal/gql"
	"github.com/pownieh/stellar_go/services/ticker/internal/tickerdb"
	hlog "github.com/pownieh/stellar_go/support/log"
)

func StartGraphQLServer(s *tickerdb.TickerSession, l *hlog.Entry, port string) {
	graphql := gql.New(s, l)

	graphql.Serve(port)
}
