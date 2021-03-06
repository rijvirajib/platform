package cmd

import (
	"os"
	"strings"

	"github.com/goodeggs/platform/cmd/ranch/Godeps/_workspace/src/golang.org/x/crypto/ssh/terminal"

	"github.com/goodeggs/platform/cmd/ranch/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/goodeggs/platform/cmd/ranch/util"
)

var execCmd = &cobra.Command{
	Use:   "exec <pid> <command>",
	Short: "Execute a command in an existing process",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		appName, err := util.AppName(cmd)
		if err != nil {
			return err
		}

		pid := args[0]
		command := strings.Join(args[1:], " ")

		exitCode, err := exec(appName, pid, command)
		if err != nil {
			return err
		}

		os.Exit(exitCode)
		return nil
	},
}

func exec(appName, pid, command string) (int, error) {
	fd := os.Stdin.Fd()

	if terminal.IsTerminal(int(fd)) {
		stdinState, err := terminal.GetState(int(fd))

		if err != nil {
			return -1, err
		}

		defer terminal.Restore(int(fd), stdinState)
	}

	return util.ConvoxExec(appName, pid, command, os.Stdin, os.Stdout)
}

func init() {
	RootCmd.AddCommand(execCmd)
}
