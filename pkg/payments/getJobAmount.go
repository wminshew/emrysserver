package payments

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"math"
)

const minJobAmt = 1

func getJobAmount(jUUID uuid.UUID) (int64, error) {
	rate, createdAt, completedAt, canceledAt, failedAt, err := db.GetJobPaymentInfo(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job payment info",
			"err", err.Error(),
			"jID", jUUID,
		)
		return 0, err
	}

	if rate == 0 || createdAt.IsZero() {
		message := "error no job rate or created_at"
		err := fmt.Errorf(message)
		log.Sugar.Errorw(message,
			"err", err.Error(),
			"jID", jUUID,
		)
		return 0, err
	}

	var amt int64
	if !completedAt.IsZero() {
		amt = int64(math.Round(rate * completedAt.Sub(createdAt).Hours() * 100))
	} else if !canceledAt.IsZero() {
		amt = int64(math.Round(rate * canceledAt.Sub(createdAt).Hours() * 100))
	} else if !failedAt.IsZero() {
		amt = int64(math.Round(rate * failedAt.Sub(createdAt).Hours() * 100))
	} else {
		message := "error no job rate or created_at"
		err := fmt.Errorf(message)
		log.Sugar.Errorw(message,
			"err", err.Error(),
			"jID", jUUID,
		)
		return 0, err
	}

	if amt < minJobAmt {
		amt = minJobAmt
	}

	return amt, nil
}
