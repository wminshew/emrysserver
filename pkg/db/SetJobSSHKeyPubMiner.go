package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetJobSSHKeyPubMiner sets job ssh_key_miner for job jUUID
func SetJobSSHKeyPubMiner(jUUID uuid.UUID, sshKeyPubMiner string) error {
	sqlStmt := `
	UPDATE jobs j
	SET ssh_key_miner = $2
	WHERE j.uuid = $1
	`
	_, err := db.Exec(sqlStmt, jUUID, sshKeyPubMiner)
	if err != nil {
		message := "error updating jobs ssh_key_miner"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
			)
		}
	}
	return err
}
