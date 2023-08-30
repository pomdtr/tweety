package cmd

import "github.com/spf13/cobra"

type Profile struct {
	Path             string
	Args             []string
	Env              map[string]string
	WorkingDirectory string
}

func NewCmdProfileList() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := sendMessage[map[string]Profile](map[string]string{
				"command": "profile.list",
			})
			if err != nil {
				return err
			}

			for key := range profiles {
				cmd.Printf("%s\n", key)
			}

			return nil
		},
	}

	return cmd
}

func NewCmdProfile() *cobra.Command {
	cmd := &cobra.Command{
		Use: "profile",
	}

	cmd.AddCommand(NewCmdProfileList())
	return cmd
}
