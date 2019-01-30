package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const keySize = 4096

// postUser handles new users for notebook service
var postUser app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	jID := r.URL.Query().Get("jID")
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	if err := addUser(jUUID); err != nil {
		log.Sugar.Errorw("error creating user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error creating user"}
	}

	homeDir := filepath.Join("home", jUUID.String())
	if err := os.Chmod(homeDir, 0700); err != nil {
		log.Sugar.Errorw("error updating user home permissions",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error creating user"}
	}

	if err := unlockUser(jUUID); err != nil {
		log.Sugar.Errorw("error unlocking user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error creating user"}
	}

	sshKeyPath := filepath.Join("ssh-keys", jUUID.String())
	sshKeyPubPath := sshKeyPath + ".pub"
	if err := createSSHKeyPair(sshKeyPath, sshKeyPubPath); err != nil {
		log.Sugar.Errorw("error creating ssh key pair",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error creating ssh key pair"}
	}
	defer func() {
		if err := deleteSSHKeyPair(sshKeyPath, sshKeyPubPath); err != nil {
			log.Sugar.Errorw("error deleting ssh key pair",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
		}
	}()
	// TODO: generate & delete ssh key for miner?

	sshKeyPubBytes, err := ioutil.ReadFile(sshKeyPubPath)
	if err != nil {
		log.Sugar.Errorw("error reading ssh pubkey file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error reading ssh pubkey file"}
	}
	sshKeyPub := string(sshKeyPubBytes)
	if err := db.SetJobSSHKeyUser(jUUID, sshKeyPub); err != nil {
		log.Sugar.Errorw("error setting job ssh pubkey for user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error setting job ssh pubkey for user"}
	}

	sshKeyBytes, err := ioutil.ReadFile(sshKeyPath)
	if err != nil {
		log.Sugar.Errorw("error reading ssh key file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error reading ssh key file"}
	}
	if _, err := w.Write(sshKeyBytes); err != nil {
		log.Sugar.Errorw("error writing job ssh key to user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error writing job ssh key to user"}
	}

	// TODO: update db status?
	return nil
}

func addUser(jUUID uuid.UUID) error {
	cmdStr := "adduser"
	args := []string{"-D", jUUID.String(), "-G", "emrys"}
	cmd := exec.Command(cmdStr, args...)
	return cmd.Run()
}

func unlockUser(jUUID uuid.UUID) error {
	cmdStr := "passwd"
	args := []string{"-u", jUUID.String()}
	cmd := exec.Command(cmdStr, args...)
	return cmd.Run()
}

func createSSHKeyPair(sshKeyPath, sshKeyPubPath string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}

	// generate and write private key as PEM
	privKeyFile, err := os.Create(sshKeyPath)
	defer check.Err(privKeyFile.Close)
	if err != nil {
		return err
	}
	privKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)}
	if err := pem.Encode(privKeyFile, privKeyPEM); err != nil {
		return err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(sshKeyPubPath, ssh.MarshalAuthorizedKey(pub), 0644)
}

func deleteSSHKeyPair(sshKeyPath, sshKeyPubPath string) error {
	if err := os.Remove(sshKeyPath); err != nil {
		return err
	}
	return os.Remove(sshKeyPubPath)
}
