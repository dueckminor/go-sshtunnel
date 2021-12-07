package commands

import (
	"fmt"
	"os"

	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/manifoldco/promptui"
)

func init() {
	RegisterCommand("connect", cmdConnect{})
}

type cmdConnect struct{}

func (cmdConnect) Execute(args ...string) error {
	c := control.Client()

	in := control.ConnectIn{}
	for {
		out, err := c.Connect(in)
		if err != nil {
			return err
		}
		in.Passphrase = ""
		in.ID = out.ID
		for _, msg := range out.Messages {
			fmt.Println(msg)
		}
		switch out.Status {
		case control.ConnectStatusFailed:
			os.Exit(1)
		case control.ConnectStatusSucceeded:
			os.Exit(0)
		case control.ConnectStatusNeedPassphrase:
			templates := &promptui.PromptTemplates{
				Prompt:  "{{ . | bold }} ",
				Valid:   "{{ . | bold }} ",
				Invalid: "{{ . | bold }} ",
				Success: "{{ . | bold }} ",
			}

			prompt := promptui.Prompt{
				Label:     "Passphrase:",
				Mask:      '\u2022',
				Templates: templates,
			}

			in.Passphrase, err = prompt.Run()
			if err != nil {
				return err
			}
		}
	}
}
