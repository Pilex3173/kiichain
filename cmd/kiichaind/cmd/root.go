package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/tx/config as authtxconfig"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"

	"kiichain/app"
	"kiichain/app/kiichain"
	"kiichain/evm/evmkeyring"

	"github.com/tendermint/tendermint/libs/log"
)


// tempDir membuat folder sementara untuk keperluan app
// Caller yang bertanggung jawab untuk menghapusnya.
var tempDir = func() string {
	dir, err := os.MkdirTemp("", ".kiichain")
	if err != nil {
		panic(fmt.Sprintf("gagal membuat folder temp: %s", err.Error()))
	}
	return dir
}

// NewRootCmd membuat root command untuk kiichaind
func NewRootCmd() *cobra.Command {
	initAppOptions := viper.New()
	temp := tempDir()
	initAppOptions.Set(flags.FlagHome, temp)

	// Buat temporary app
	tempApp := kiichain.NewKiichainApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		temp,
		initAppOptions,
		kiichain.EmptyWasmOptions,
		kiichain.EVMAppOptions,
	)

	defer func() {
		_ = tempApp.Close()
		if temp != kiichain.DefaultNodeHome {
			_ = os.RemoveAll(temp) // cleanup manual
		}
	}()

	// Siapkan initial client context
	initClientCtx := client.Context{}.
		WithCodec(tempApp.AppCodec()).
		WithInterfaceRegistry(tempApp.InterfaceRegistry()).
		WithLegacyAmino(tempApp.LegacyAmino()).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(kiichain.DefaultNodeHome).
		WithViper("").
		WithKeyringOptions(evmkeyring.Option()).
		WithLedgerHasProtobuf(true)

	rootCmd := &cobra.Command{
		Use:   "kiichaind",
		Short: "Kiichain App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			var err error
			initClientCtx, err = client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if !initClientCtx.Offline {
				txConfigOpts := tx.ConfigOptions{
					EnabledSignModes:           append(tx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL),
					TextualCoinMetadataQueryFn: authtxconfig.NewGRPCCoinMetadataQueryFn(initClientCtx),
				}
				txConfigWithTextual, err := tx.NewTxConfigWithOptions(initClientCtx.Codec, txConfigOpts)
				if err != nil {
					return err
				}
				initClientCtx = initClientCtx.WithTxConfig(txConfigWithTextual)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			appTemplate, appConfig := initAppConfig(kiichain.KiichainID)
			cometCfg := initCometConfig()

			return server.InterceptConfigsPreRunHandler(cmd, appTemplate, appConfig, cometCfg)
		},
	}

	// Tambah subcommands
	initRootCmd(rootCmd, tempApp.ModuleBasics, tempApp.AppCodec(), tempApp.InterfaceRegistry(), tempApp.GetTxConfig())

	// Auto CLI enhancement
	if err := enrichAutoCliOpts(tempApp.AutoCliOpts(), initClientCtx).EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd
}

// appExport untuk export state dari app
func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore string,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
) (servertypes.ExportedApp, error) {
	var kiichainApp *kiichain.KiichainApp

	if height != -1 {
		kiichainApp = kiichain.NewKiichainApp(logger, db, traceStore, false, map[int64]bool{}, "", appOpts, nil, nil)
		if err := kiichainApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		kiichainApp = kiichain.NewKiichainApp(logger, db, traceStore, true, map[int64]bool{}, "", appOpts, nil, nil)
	}

	return kiichainApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs)
}

// overrideEVMChainID jika chain-id default
func overrideEVMChainID(ctx client.Context, chainID string) client.Context {
	if ctx.ChainID == "" || ctx.ChainID == kiichain.KiichainID {
		ctx = ctx.WithChainID(chainID)
	}
	return ctx
}

// displayInfo cetak info dalam JSON
func displayInfo(info interface{}) {
	out, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(os.Stdout, string(out))
}
