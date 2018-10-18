package payments

import (
	"github.com/wminshew/emrysserver/pkg/log"
)

// ChargeUser charges the user for job jUUID
func ChargeUser(r *http.Request, jUUID uuid.UUID) error {
	// TODO: stripe or btcpay
	return db.SetJobPaidByUser(r, jUUID)
}
