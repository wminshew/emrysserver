package email

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/wminshew/emrysserver/pkg/log"
)

const (
	// TODO: move to ENV
	payoutFailedTemplateID = "d-9e0af9d2bf37450fa28a7be1a1ef1708"
)

// SendPayoutFailed sends an email to the support@emrys.io that a payout failed
func SendPayoutFailed(dest, amt string) error {
	m := mail.NewV3Mail()

	e := mail.NewEmail(fromName, fromAddress)
	m.SetFrom(e)

	m.SetTemplateID(payoutFailedTemplateID)

	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail("support", supportEmail),
	}
	p.AddTos(tos...)

	p.SetDynamicTemplateData("dest", dest)
	p.SetDynamicTemplateData("amt", amt)

	m.AddPersonalizations(p)

	request := sendgrid.GetRequest(sendgridSecret, sendgridPath, sendgridHost)
	request.Method = "POST"
	Body := mail.GetRequestBody(m)
	request.Body = Body
	response, err := sendgrid.API(request)
	if err != nil {
		return err
	}

	// TODO: remove? handle non 2xx status codes?
	log.Sugar.Infow("payout-failed email sent to support",
		"StatusCode", response.StatusCode,
		"Body", response.Body,
		"Headers", response.Headers,
	)

	return nil
}
