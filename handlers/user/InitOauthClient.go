package user

import (
	"github.com/wminshew/emrysserver/pkg/app"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	credentialsPath = os.Getenv("CLOUDBUILDER_CREDENTIAL_FILE")
	oauthClient     *http.Client
)

// InitOauthClient initializes an http client authorized with google via oauth2
func InitOauthClient() {
	app.Sugar.Infof("Initializing google oauth client...")

	ctx := oauth2.NoContext
	data, _ := ioutil.ReadFile(credentialsPath)
	creds, err := google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		app.Sugar.Errorf("oauthClient failed to initialize! Panic!")
		panic(err)
	}
	oauthClient = oauth2.NewClient(ctx, creds.TokenSource)
}
