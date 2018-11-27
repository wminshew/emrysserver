package payments

// import (
// 	"github.com/satori/go.uuid"
// 	"github.com/wminshew/emrysserver/pkg/db"
// 	"github.com/wminshew/emrysserver/pkg/log"
// 	"net/http"
// )

// ChargeUsers charges users for their outstanding balances
func ChargeUsers() {
	// func ChargeUser(r *http.Request, jUUID uuid.UUID) error {
	// TODO: trigger on weekly basis (like miner payouts)
	// TODO: make atomic db transaction: get user balance, charge w/ stripe, add tx to payment log, update user balance

	// TODO: add stripe

	// TODO: add amount?
	// log.Sugar.Infow("user charged",
	// 	"method", r.Method,
	// 	"url", r.URL,
	// 	"jID", jUUID,
	// )
	//
	// return db.SetJobPaidByUser(r, jUUID)
}
