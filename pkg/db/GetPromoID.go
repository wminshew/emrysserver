package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetPromoID returns the ID of a promo
func GetPromoID(promo string) (int, error) {
	var nullPromoID sql.NullInt64
	sqlStmt := `
	SELECT id
	FROM promos
	WHERE promo = $1
	`
	if err := db.QueryRow(sqlStmt, promo).Scan(&nullPromoID); err != nil {
		message := "error querying promo ID"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"promo", promo,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"promo", promo,
			)
		}
		return 0, err
	}
	var promoID int
	if nullPromoID.Valid {
		promoID = int(nullPromoID.Int64)
	}
	// TODO: returns 0 if nullPromoID is invalid, but that should never happen... throw panic?
	return promoID, nil
}
