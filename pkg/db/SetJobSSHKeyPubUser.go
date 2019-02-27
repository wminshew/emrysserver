package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetJobSSHKeyPubUser sets job ssh_key_user for job jUUID
func SetJobSSHKeyPubUser(jUUID uuid.UUID, sshKeyPubUser string) error {
	sqlStmt := `
	UPDATE jobs j
	SET ssh_key_user = $2
	WHERE j.uuid = $1
	`
	_, err := db.Exec(sqlStmt, jUUID, sshKeyPubUser)
	if err != nil {
		message := "error updating jobs ssh_key_user"
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
	}
	return err
}
