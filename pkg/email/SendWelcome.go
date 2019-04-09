package email

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/wminshew/emrysserver/pkg/log"
)

const (
	// TODO: move to ENV
	welcomeTemplateID = "d-b92233a304484298ac2ffb85c4b307b1"
)

// SendWelcome sends a welcome email to the user's address
func SendWelcome(email string) error {
	m := mail.NewV3Mail()

	e := mail.NewEmail(fromName, fromAddress)
	m.SetFrom(e)

	m.SetTemplateID(welcomeTemplateID)

	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(email, email),
	}
	p.AddTos(tos...)

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
	log.Sugar.Infow("user welcome email sent",
		"StatusCode", response.StatusCode,
		"Body", response.Body,
		"Headers", response.Headers,
	)

	return nil
}
