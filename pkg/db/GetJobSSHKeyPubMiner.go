package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetJobSSHKeyPubMiner gets ssh_key_miner for job jUUID
func GetJobSSHKeyPubMiner(jUUID uuid.UUID) (string, error) {
	var sshKey sql.NullString
	sqlStmt := `
	SELECT j.ssh_key_miner
	FROM jobs j
	WHERE j.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&sshKey); err != nil {
		message := "error getting job ssh_key_miner"
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
		return "", err
	}
	if sshKey.Valid {
		return sshKey.String, nil
	}
	return "", nil
}
