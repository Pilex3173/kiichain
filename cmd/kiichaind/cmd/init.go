package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/module"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/kiichain/kiichain/app/params"
	"github.com/tendermint/tendermint/config"
)

func initCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize node configuration (validator, p2p, genesis)",
		Long: `The init command sets up configuration files for your node, including
the private validator key, P2P networking, and the genesis file.`,
		Example: `
  # Init node with a custom moniker and chain-id
  kiichaind init mynode --chain-id kiichain-1

  # Init node using a recovery seed phrase
  kiichaind init mynode --recover

  # Init and overwrite existing genesis.json
  kiichaind init mynode --overwrite
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initializes the client context from the CMD and the codecs
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			// ... bagian original init logic di sini (generate key, nodeID, genesis, dsb) ...

			// Print the chain info
			toPrint := newPrintInfo(config.Moniker, chainID, nodeID, "", appState)

			// Set the custom chain config
			params.SetTendermintConfigs(config)

			// Save config.toml
			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)

			// Check output format
			output, _ := cmd.Flags().GetString("output")
			return displayInfo(toPrint, output)
		},
	}

	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(genutilcli.FlagDefaultBondDenom, "", "genesis file default denomination, if left blank default value is 'akii'")
	cmd.Flags().Int64(flags.FlagInitHeight, 1, "specify the initial block height at genesis")

	// Add output format flag
	cmd.Flags().String("output", "json", "Output format (json|text)")

	return cmd
}

// displayInfo formats and prints the information in a user-friendly way
func displayInfo(info printInfo, format string) error {
	switch format {
	case "text":
		fmt.Printf("\n✅ Node initialized successfully!\n")
		fmt.Printf("🔹 Moniker: %s\n", info.Moniker)
		fmt.Printf("🔹 Chain ID: %s\n", info.ChainID)
		fmt.Printf("🔹 Node ID: %s\n", info.NodeID)
		return nil
	default:
		out, err := json.MarshalIndent(info, "", " ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(os.Stderr, "%s\n", out)
		return err
	}
}
