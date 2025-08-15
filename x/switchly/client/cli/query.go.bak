package cli

import (
	"log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/switchlyprotocol/switchlynode/v3/common/relay"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

type ver struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

func (v ver) String() string {
	return v.Version
}

func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "switchly",
		Short:                      "Querying commands for the switchly module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	return queryCmd
}

// GetCmdGetVersion queries current version
func GetCmdGetVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Gets the Switchly version and build information",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			clientCtx.OutputFormat = "json"

			out := ver{
				Version:   constants.SWVersion.String(),
				GitCommit: constants.GitCommit,
				BuildTime: constants.BuildTime,
			}
			return clientCtx.PrintObjectLegacy(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetCmdGetNORelay() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discord-relay",
		Short: "Relays a message from a node operator to discord",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			msg := relay.NewNodeRelay(args[0], args[1])

			if err = msg.Prepare(); err != nil {
				log.Fatalln(err)
			}

			result, err := msg.Broadcast()
			if err != nil {
				log.Fatalln(err)
			}

			return clientCtx.PrintObjectLegacy(result)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
