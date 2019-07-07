package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

// GetPromoInfo gets information about a promo code
func GetPromoInfo(promo string) (int, time.Time, int, int, error) {
	var nullCredit, nullUses, nullMaxUses sql.NullInt64
	var nullExpiration pq.NullTime
	sqlStmt := `
	SELECT credit, expiration, uses, max_uses
	FROM promos
	WHERE promo = $1
	`
	if err := db.QueryRow(sqlStmt, promo).Scan(&nullCredit, &nullExpiration, &nullUses, &nullMaxUses); err != nil {
		message := "error querying promo info"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
			)
		}
		return 0, time.Time{}, 0, 0, err
	}
	var credit, uses, maxUses int
	expiration := time.Time{}

	if nullCredit.Valid {
		credit = int(nullCredit.Int64)
	}
	if nullUses.Valid {
		uses = int(nullUses.Int64)
	}
	if nullMaxUses.Valid {
		maxUses = int(nullMaxUses.Int64)
	}
	if nullExpiration.Valid {
		expiration = nullExpiration.Time
	}

	return credit, expiration, uses, maxUses, nil
}
