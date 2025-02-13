package email

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/wminshew/emrysserver/pkg/log"
)

const (
	// TODO: move to ENV
	registrationTemplateID = "d-5bfe76c1b2a94855980f74f8cc6bc205"
)

// SendEmailConfirmation sends an email to the user's address to confirm ownership
func SendEmailConfirmation(email, token string) error {
	m := mail.NewV3Mail()

	e := mail.NewEmail(fromName, fromAddress)
	m.SetFrom(e)

	m.SetTemplateID(registrationTemplateID)

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
	log.Sugar.Infow("user registration email confirmation sent",
		"StatusCode", response.StatusCode,
		"Body", response.Body,
		"Headers", response.Headers,
	)

	return nil
}
