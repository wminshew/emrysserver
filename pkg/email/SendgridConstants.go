package email

import (
	"os"
)

var sendgridSecret = os.Getenv("SENDGRID_SECRET")

const (
	fromName     = "emrys"
	fromAddress  = "support@emrys.io"
	supportEmail = "support@emrys.io"
	sendgridPath = "/v3/mail/send"
	sendgridHost = "https://api.sendgrid.com"
)
