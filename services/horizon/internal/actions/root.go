package actions

import (
	"net/http"
	"net/url"

	"github.com/pownieh/stellar_go/services/horizon/internal/ledger"
	"github.com/pownieh/stellar_go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/protocols/horizon"
)

type GetRootHandler struct {
	LedgerState *ledger.State
	CoreStateGetter
	NetworkPassphrase string
	FriendbotURL      *url.URL
	HorizonVersion    string
}

func (handler GetRootHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	var res horizon.Root
	templates := map[string]string{
		"accounts":           AccountsQuery{}.URITemplate(),
		"claimableBalances":  ClaimableBalancesQuery{}.URITemplate(),
		"liquidityPools":     LiquidityPoolsQuery{}.URITemplate(),
		"offers":             OffersQuery{}.URITemplate(),
		"strictReceivePaths": StrictReceivePathsQuery{}.URITemplate(),
		"strictSendPaths":    FindFixedPathsQuery{}.URITemplate(),
	}
	coreState := handler.GetCoreState()
	resourceadapter.PopulateRoot(
		r.Context(),
		&res,
		handler.LedgerState.CurrentStatus(),
		handler.HorizonVersion,
		coreState.CoreVersion,
		handler.NetworkPassphrase,
		coreState.CurrentProtocolVersion,
		coreState.CoreSupportedProtocolVersion,
		handler.FriendbotURL,
		templates,
	)
	return res, nil
}
