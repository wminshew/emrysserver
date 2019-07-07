package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// InsertPromoUse inserts a new promo use into the db
func InsertPromoUse(aUUID uuid.UUID, promo string) error {
	promoID, err := GetPromoID(promo)
	if err != nil {
		return err // already logged
	}

	sqlStmt := `
	INSERT INTO promos_log (account_uuid, promo_id)
	VALUES ($1, $2)
	`
	if _, err := db.Exec(sqlStmt, aUUID, promoID); err != nil {
		message := "error inserting promo use"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"aID", aUUID,
				"promo", promo,
				"promo_id", promoID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"aID", aUUID,
				"promo", promo,
				"promo_id", promoID,
			)
		}
		return err
	}
	return nil
}
