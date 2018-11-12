package email

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/wminshew/emrysserver/pkg/log"
)

const (
	resetPasswordTemplateID = "d-9f09c620fc724ea4ba20f248b48f6d37"
)

// SendResetPasswordsends an email to the user's address to reset their password
func SendResetPassword(email, token string) error {
	m := mail.NewV3Mail()

	e := mail.NewEmail(fromName, fromAddress)
	m.SetFrom(e)

	m.SetTemplateID(resetPasswordTemplateID)

	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(email, email),
	}
	p.AddTos(tos...)

	p.SetDynamicTemplateData("token", token)

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
	log.Sugar.Infow("reset password email sent",
		"StatusCode", response.StatusCode,
		"Body", response.Body,
		"Headers", response.Headers,
	)

	return nil
}
