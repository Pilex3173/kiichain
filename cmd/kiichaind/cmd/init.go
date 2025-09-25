package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	tmconfig "github.com/cometbft/cometbft/config"
	tmtypes "github.com/cometbft/cometbft/types"
	p2p "github.com/cometbft/cometbft/p2p"
)

// printInfo struct untuk hasil init
type printInfo struct {
	Moniker string          `json:"moniker"`
	ChainID string          `json:"chain_id"`
	NodeID  string          `json:"node_id"`
	AppState json.RawMessage `json:"app_state"`
}

func newPrintInfo(moniker, chainID, nodeID string, appState json.RawMessage) printInfo {
	return printInfo{
		Moniker:  moniker,
		ChainID:  chainID,
		NodeID:   nodeID,
		AppState: appState,
	}
}

// InitCmd perintah untuk inisialisasi node
func InitCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator, p2p node ID, genesis, and config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			// Load default config
			config := tmconfig.DefaultConfig()
			config.SetRoot(clientCtx.HomeDir)

			// Ambil NodeID
			nodeKey, err := p2p.LoadOrGenNodeKey(filepath.Join(config.RootDir, "config", "node_key.json"))
			if err != nil {
				return fmt.Errorf("failed to load or gen node_key: %w", err)
			}
			nodeID := string(nodeKey.ID())

			// Dummy app state kosong
			appState := json.RawMessage(`{}`)

			// Ambil moniker & chain-id dari CLI
			config.Moniker = args[0]
			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			if chainID != "" {
				clientCtx = clientCtx.WithChainID(chainID)
			}

			// Tulis genesis.json
			genFile := filepath.Join(config.RootDir, "config", "genesis.json")
			genDoc := tmtypes.GenesisDoc{
				ChainID:  clientCtx.ChainID,
				AppState: appState,
			}
			if err := genDoc.SaveAs(genFile); err != nil {
				return fmt.Errorf("failed to write genesis file: %w", err)
			}

			// Tulis config.toml
			tmconfig.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)

			// Print hasil
			toPrint := newPrintInfo(config.Moniker, clientCtx.ChainID, nodeID, appState)
			output, _ := cmd.Flags().GetString("output")
			return displayInfo(toPrint, output)
		},
	}

	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id")
	cmd.Flags().String("output", "text", "Output format (text|json)")

	return cmd
}

// displayInfo tampilkan hasil init
func displayInfo(info printInfo, format string) error {
	switch format {
	case "text":
		fmt.Printf("\n✅ Node initialized successfully!\n")
		fmt.Printf("🔹 Moniker: %s\n", info.Moniker)
		fmt.Printf("🔹 Chain ID: %s\n", info.ChainID)
		fmt.Printf("🔹 Node ID: %s\n", info.NodeID)
		return nil
	default:
		out, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(os.Stdout, "%s\n", out) // ✅ ke stdout
		return err
	}
}
