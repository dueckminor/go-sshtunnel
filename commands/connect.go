package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/manifoldco/promptui"
)

func init() {
	RegisterCommand("connect", cmdConnect{})
}

type cmdConnect struct{}

func (cmdConnect) Execute(args ...string) error {
	acceptHostKeys := false
	filteredArgs := args[:0]
	for _, a := range args {
		if a == "--accept-host-keys" {
			acceptHostKeys = true
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}
	_ = filteredArgs

	c := control.Client()

	in := control.ConnectIn{}
	for {
		out, err := c.Connect(in)
		if err != nil {
			return err
		}
		in.Passphrase = ""
		in.AcceptHostKey = nil
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
		case control.ConnectStatusUnknownHostKey:
			if acceptHostKeys {
				fmt.Printf("Auto-accepting host key (fingerprint: %s)\n", out.HostKeyFingerprint)
				accept := true
				in.AcceptHostKey = &accept
			} else {
				fmt.Printf("Are you sure you want to continue connecting (yes/no)? ")

				reader := bufio.NewReader(os.Stdin)
				answer, readErr := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))

				accept := readErr == nil && (answer == "yes" || answer == "y")
				in.AcceptHostKey = &accept
				if !accept {
					fmt.Println("Host key rejected.")
					os.Exit(1)
				}
				fmt.Println("Host key accepted and will be added to known_hosts.")
			}
		}
	}
}
