package payments

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// ChargeUser charges the user for job jUUID
func ChargeUser(r *http.Request, jUUID uuid.UUID) error {
	// TODO: stripe or btcpay

	// TODO: add amount?
	log.Sugar.Infow("user charged",
		"method", r.Method,
		"url", r.URL,
		"jID", jUUID,
	)

	return db.SetJobPaidByUser(r, jUUID)
}
