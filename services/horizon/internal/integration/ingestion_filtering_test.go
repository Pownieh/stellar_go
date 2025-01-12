package integration

import (
	"strconv"
	"testing"
	"time"

	"github.com/pownieh/stellar_go/clients/horizonclient"
	hProtocol "github.com/pownieh/stellar_go/protocols/horizon"
	"github.com/pownieh/stellar_go/services/horizon/internal/ingest/filters"
	"github.com/pownieh/stellar_go/services/horizon/internal/test/integration"
	"github.com/pownieh/stellar_go/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestFilteringWithNoFilters(t *testing.T) {
	tt := assert.New(t)
	const adminPort uint16 = 6000
	itest := integration.NewTest(t, integration.Config{
		HorizonIngestParameters: map[string]string{
			"admin-port": strconv.Itoa(int(adminPort)),
		},
	})

	fullKeys, accounts := itest.CreateAccounts(2, "10000")
	nonWhitelistedAccount := accounts[1]
	nonWhitelistedAccountKey := fullKeys[1]

	// all assets are allowed by default because the asset filter config is empty.
	defaultAllowedAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(nonWhitelistedAccountKey, nonWhitelistedAccount, defaultAllowedAsset)

	// Assert that by default, the system allows all the accounts.
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: nonWhitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       defaultAllowedAsset,
		},
	)
	txResp, err := itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)
}

func TestFilteringAccountWhiteList(t *testing.T) {
	tt := assert.New(t)
	const adminPort uint16 = 6000
	itest := integration.NewTest(t, integration.Config{
		HorizonIngestParameters: map[string]string{
			"admin-port": strconv.Itoa(int(adminPort)),
		},
	})

	fullKeys, accounts := itest.CreateAccounts(2, "10000")
	whitelistedAccount := accounts[0]
	whitelistedAccountKey := fullKeys[0]
	nonWhitelistedAccount := accounts[1]
	nonWhitelistedAccountKey := fullKeys[1]
	enabled := true

	// all assets are allowed by default because the asset filter config is empty.
	defaultAllowedAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(whitelistedAccountKey, whitelistedAccount, defaultAllowedAsset)
	itest.MustEstablishTrustline(nonWhitelistedAccountKey, nonWhitelistedAccount, defaultAllowedAsset)

	// Setup a whitelisted account rule, force refresh of filter configs to be quick
	filters.SetFilterConfigCheckIntervalSeconds(1)

	expectedAccountFilter := hProtocol.AccountFilterConfig{
		Whitelist: []string{whitelistedAccount.GetAccountID()},
		Enabled:   &enabled,
	}
	err := itest.AdminClient().SetIngestionAccountFilter(expectedAccountFilter)
	tt.NoError(err)

	accountFilter, err := itest.AdminClient().GetIngestionAccountFilter()
	tt.NoError(err)

	tt.ElementsMatch(expectedAccountFilter.Whitelist, accountFilter.Whitelist)
	tt.Equal(expectedAccountFilter.Enabled, accountFilter.Enabled)

	// Ensure the latest filter configs are reloaded by the ingestion state machine processor
	time.Sleep(time.Duration(filters.GetFilterConfigCheckIntervalSeconds()) * time.Second)

	// Make sure that when using a non-whitelisted account, the transaction is not stored
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: nonWhitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       defaultAllowedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.True(horizonclient.IsNotFoundError(err))

	// Make sure that when using a whitelisted account, the transaction is stored
	txResp = itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: whitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       defaultAllowedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)
}

func TestFilteringAssetWhiteList(t *testing.T) {
	tt := assert.New(t)
	const adminPort uint16 = 6000
	itest := integration.NewTest(t, integration.Config{
		HorizonIngestParameters: map[string]string{
			"admin-port": strconv.Itoa(int(adminPort)),
		},
	})

	fullKeys, accounts := itest.CreateAccounts(1, "10000")
	defaultAllowedAccount := accounts[0]
	defaultAllowedAccountKey := fullKeys[0]

	whitelistedAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(defaultAllowedAccountKey, defaultAllowedAccount, whitelistedAsset)

	nonWhitelistedAsset := txnbuild.CreditAsset{Code: "SEK", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(defaultAllowedAccountKey, defaultAllowedAccount, nonWhitelistedAsset)
	enabled := true

	// Setup a whitelisted asset rule, force refresh of filters to be quick
	filters.SetFilterConfigCheckIntervalSeconds(1)

	asset, err := whitelistedAsset.ToXDR()
	tt.NoError(err)
	expectedAssetFilter := hProtocol.AssetFilterConfig{
		Whitelist: []string{asset.StringCanonical()},
		Enabled:   &enabled,
	}
	err = itest.AdminClient().SetIngestionAssetFilter(expectedAssetFilter)
	tt.NoError(err)

	assetFilter, err := itest.AdminClient().GetIngestionAssetFilter()
	tt.NoError(err)

	tt.ElementsMatch(expectedAssetFilter.Whitelist, assetFilter.Whitelist)
	tt.Equal(expectedAssetFilter.Enabled, assetFilter.Enabled)

	// Ensure the latest filter configs are reloaded by the ingestion state machine processor
	time.Sleep(time.Duration(filters.GetFilterConfigCheckIntervalSeconds()) * time.Second)

	// Make sure that when using a non-whitelisted asset, the transaction is not stored
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: defaultAllowedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       nonWhitelistedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.True(horizonclient.IsNotFoundError(err))

	// Make sure that when using a whitelisted asset, the transaction is stored
	txResp = itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: defaultAllowedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       whitelistedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)
}
