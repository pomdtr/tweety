package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	_ "embed"

	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

func main() {
	var flags struct {
		host string
		port int
	}
	cmd := cobra.Command{
		Use:          "tweety",
		Short:        "An integrated terminal for your web browser",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := NewHandler()
			if err != nil {
				return err
			}

			port := flags.port
			if port == 0 {
				p, err := freeport.GetFreePort()
				if err != nil {
					return err
				}

				port = p
			}

			browserUrl, _ := url.Parse("https://tweety.sh")
			if cmd.Flags().Changed("port") {
				query := browserUrl.Query()
				query.Set("port", fmt.Sprintf("%d", port))
				browserUrl.RawQuery = query.Encode()
			}

			cmd.PrintErrln("Listening on", fmt.Sprintf("http://%s:%d", flags.host, port))
			cmd.PrintErrln("Browser Friendly URL:", browserUrl.String())
			cmd.Println("Press Ctrl+C to exit")
			return http.ListenAndServe(fmt.Sprintf("%s:%d", flags.host, port), handler)
		},
	}

	cmd.Flags().StringVarP(&flags.host, "host", "H", "localhost", "host to listen on")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 9999, "port to listen on")

	cmd.AddCommand(NewCompletionCmd(cmd.Name()))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewCompletionCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: fmt.Sprintf(`To load completions:

	Bash:

	  $ source <(%[1]s completion bash)

	  # To load completions for each session, execute once:
	  # Linux:
	  $ %[1]s completion bash > /etc/bash_completion.d/%[1]s
	  # macOS:
	  $ %[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s

	Zsh:

	  # If shell completion is not already enabled in your environment,
	  # you will need to enable it.  You can execute the following once:

	  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

	  # To load completions for each session, execute once:
	  $ %[1]s completion zsh > "${fpath[1]}/_%[1]s"

	  # You will need to start a new shell for this setup to take effect.

	fish:

	  $ %[1]s completion fish | source

	  # To load completions for each session, execute once:
	  $ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

	PowerShell:

	  PS> %[1]s completion powershell | Out-String | Invoke-Expression

	  # To load completions for every new session, run:
	  PS> %[1]s completion powershell > %[1]s.ps1
	  # and source this file from your PowerShell profile.
	`, name),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}
