package serve

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pownieh/stellar_go/keypair"
	"github.com/pownieh/stellar_go/strkey"
	supportlog "github.com/pownieh/stellar_go/support/log"
	"github.com/pownieh/stellar_go/support/render/httpjson"
	"github.com/pownieh/stellar_go/txnbuild"
)

// ChallengeHandler implements the SEP-10 challenge endpoint and handles
// requests for a new challenge transaction.
type challengeHandler struct {
	Logger             *supportlog.Entry
	NetworkPassphrase  string
	SigningKey         *keypair.Full
	ChallengeExpiresIn time.Duration
	Domain             string
	HomeDomains        []string
}

type challengeResponse struct {
	Transaction       string `json:"transaction"`
	NetworkPassphrase string `json:"network_passphrase"`
}

func (h challengeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryValues := r.URL.Query()

	account := queryValues.Get("account")
	isStellarAccount := strkey.IsValidEd25519PublicKey(account)
	isMuxedAccount := strkey.IsValidMuxedAccountEd25519PublicKey(account)
	if !isStellarAccount && !isMuxedAccount {
		badRequest.Render(w)
		return
	}

	homeDomain := queryValues.Get("home_domain")
	if homeDomain != "" {
		// In some cases the full stop (period) character is used at the end of a FQDN.
		homeDomain = strings.TrimSuffix(homeDomain, ".")
		matched := false
		for _, supportedDomain := range h.HomeDomains {
			if homeDomain == supportedDomain {
				matched = true
				break
			}
		}
		if !matched {
			badRequest.Render(w)
			return
		}
	} else {
		homeDomain = h.HomeDomains[0]
	}

	var memo *txnbuild.MemoID
	memoParam := queryValues.Get("memo")
	if memoParam != "" {
		memoInt, err := strconv.ParseUint(memoParam, 10, 64)
		if err != nil {
			badRequest.Render(w)
			return
		}
		memoId := txnbuild.MemoID(memoInt)
		memo = &memoId
	}

	tx, err := txnbuild.BuildChallengeTx(
		h.SigningKey.Seed(),
		account,
		h.Domain,
		homeDomain,
		h.NetworkPassphrase,
		h.ChallengeExpiresIn,
		memo,
	)
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		badRequest.Render(w)
		return
	}

	hash, err := tx.HashHex(h.NetworkPassphrase)
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	l := h.Logger.Ctx(ctx).
		WithField("tx", hash).
		WithField("account", account).
		WithField("serversigner", h.SigningKey.Address()).
		WithField("homedomain", homeDomain)

	l.Info("Generated challenge transaction for account.")

	txeBase64, err := tx.Base64()
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	res := challengeResponse{
		Transaction:       txeBase64,
		NetworkPassphrase: h.NetworkPassphrase,
	}
	httpjson.Render(w, res, httpjson.JSON)
}
