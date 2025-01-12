package serve

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pownieh/stellar_go/clients/horizonclient"
	"github.com/pownieh/stellar_go/keypair"
	"github.com/pownieh/stellar_go/support/http/httpdecode"
	supportlog "github.com/pownieh/stellar_go/support/log"
	"github.com/pownieh/stellar_go/support/render/httpjson"
	"github.com/pownieh/stellar_go/txnbuild"
	"github.com/pownieh/stellar_go/xdr"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type tokenHandler struct {
	Logger                      *supportlog.Entry
	HorizonClient               horizonclient.ClientInterface
	NetworkPassphrase           string
	SigningAddresses            []*keypair.FromAddress
	JWK                         jose.JSONWebKey
	JWTIssuer                   string
	JWTExpiresIn                time.Duration
	AllowAccountsThatDoNotExist bool
	Domain                      string
	HomeDomains                 []string
}

type tokenRequest struct {
	Transaction string `json:"transaction" form:"transaction"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (h tokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := tokenRequest{}

	err := httpdecode.Decode(r, &req)
	if err != nil {
		badRequest.Render(w)
		return
	}

	var (
		tx              *txnbuild.Transaction
		clientAccountID string
		signingAddress  *keypair.FromAddress
		homeDomain      string
		memo            *txnbuild.MemoID
	)
	for _, s := range h.SigningAddresses {
		tx, clientAccountID, homeDomain, memo, err = txnbuild.ReadChallengeTx(req.Transaction, s.Address(), h.NetworkPassphrase, h.Domain, h.HomeDomains)
		if err == nil {
			signingAddress = s
			break
		}
	}
	if signingAddress == nil {
		badRequest.Render(w)
		return
	}
	if homeDomain == "" {
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
		WithField("account", clientAccountID).
		WithField("serversigner", signingAddress.Address()).
		WithField("homedomain", homeDomain).
		WithField("memo", memo)

	l.Info("Start verifying challenge transaction.")

	muxedAccount, err := xdr.AddressToMuxedAccount(clientAccountID)
	if err != nil {
		badRequest.Render(w)
		return
	}
	if muxedAccount.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		clientAccountID = muxedAccount.ToAccountId().Address()
	}

	var clientAccountExists bool
	clientAccount, err := h.HorizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: clientAccountID})
	switch {
	case err == nil:
		clientAccountExists = true
		l.Infof("Account exists.")
	case horizonclient.IsNotFoundError(err):
		clientAccountExists = false
		l.Infof("Account does not exist.")
	default:
		l.WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	var signersVerified []string
	if clientAccountExists {
		requiredThreshold := txnbuild.Threshold(clientAccount.Thresholds.HighThreshold)
		clientSignerSummary := clientAccount.SignerSummary()
		signersVerified, err = txnbuild.VerifyChallengeTxThreshold(req.Transaction, signingAddress.Address(), h.NetworkPassphrase, h.Domain, h.HomeDomains, requiredThreshold, clientSignerSummary)
		if err != nil {
			l.
				WithField("signersCount", len(clientSignerSummary)).
				WithField("signaturesCount", len(tx.Signatures())).
				WithField("requiredThreshold", requiredThreshold).
				Info("Failed to verify with signers that do not meet threshold.")
			unauthorized.Render(w)
			return
		}
	} else {
		if !h.AllowAccountsThatDoNotExist {
			l.Infof("Failed to verify because accounts that do not exist are not allowed.")
			unauthorized.Render(w)
			return
		}
		signersVerified, err = txnbuild.VerifyChallengeTxSigners(req.Transaction, signingAddress.Address(), h.NetworkPassphrase, h.Domain, h.HomeDomains, clientAccountID)
		if err != nil {
			l.Infof("Failed to verify with account master key as signer.")
			unauthorized.Render(w)
			return
		}
	}

	l.
		WithField("signers", strings.Join(signersVerified, ",")).
		Infof("Successfully verified challenge transaction.")

	jwsOptions := &jose.SignerOptions{}
	jwsOptions.WithType("JWT")
	jws, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(h.JWK.Algorithm), Key: h.JWK.Key}, jwsOptions)
	if err != nil {
		l.WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	var sub string
	if muxedAccount.Type == xdr.CryptoKeyTypeKeyTypeEd25519 {
		sub = clientAccountID
		if memo != nil {
			sub += ":" + strconv.FormatUint(uint64(*memo), 10)
		}
	} else {
		sub = muxedAccount.Address()
	}

	issuedAt := time.Unix(tx.Timebounds().MinTime, 0)
	claims := jwt.Claims{
		Issuer:   h.JWTIssuer,
		Subject:  sub,
		IssuedAt: jwt.NewNumericDate(issuedAt),
		Expiry:   jwt.NewNumericDate(issuedAt.Add(h.JWTExpiresIn)),
	}
	tokenStr, err := jwt.Signed(jws).Claims(claims).CompactSerialize()
	if err != nil {
		l.WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	res := tokenResponse{
		Token: tokenStr,
	}
	httpjson.Render(w, res, httpjson.JSON)
}
