package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/dueckminor/go-sshtunnel/dialer"

	"github.com/manifoldco/promptui"
)

func init() {
	RegisterCommand("add-ssh-key", cmdAddSSHKey{})
}

type cmdAddSSHKey struct{}

func (cmdAddSSHKey) Execute(args ...string) error {
	encodedKey := ""
	passPhrase := ""
	if len(args) > 0 {
		fileName := args[0]
		fmt.Println("Adding SSH-Key from file:", fileName)

		body, err := ioutil.ReadFile(fileName)
		if err != nil {
			panic(err)
		}
		encodedKey = string(body)

		if len(args) > 1 {
			passPhrase = args[1]
		}

		allowInteractive := len(passPhrase) == 0

		for {
			err = dialer.CheckSSHKey(encodedKey, passPhrase)
			if err == nil {
				break
			}

			if !strings.HasPrefix(err.Error(), "bcrypt_pbkdf:") &&
				err.Error() != "sshkeys: Invalid Passphrase" {
				panic(err)
			}

			if !allowInteractive {
				os.Exit(1)
			}

			templates := &promptui.PromptTemplates{
				Prompt:  "{{ . | bold }} ",
				Valid:   "{{ . | bold }} ",
				Invalid: "{{ . | bold }} ",
				Success: "{{ . | bold }} ",
			}

			prompt := promptui.Prompt{
				Label:     "Password:",
				Mask:      '\u2022',
				Templates: templates,
			}

			passPhrase, err = prompt.Run()
			if err != nil {
				return err
			}
		}

		control.Client().AddSSHKey(encodedKey, passPhrase)
	}
	return nil
}
