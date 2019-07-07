package db

import (
	"github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetPromoUses sets promos uses
func SetPromoUses(promo string, uses int) error {
	sqlStmt := `
		UPDATE promos
		SET uses = $2,
		WHERE promo = $1
		`
	if _, err := db.Exec(sqlStmt, promo, uses); err != nil {
		message := "error updating promos uses"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"promo", promo,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"promo", promo,
			)
		}
		return err
	}

	return nil
}
