package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcfg "github.com/cometbft/cometbft/config"
	tmcli "github.com/cometbft/cometbft/libs/cli"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/snapshots"
	snapshottypes "cosmossdk.io/store/snapshots/types"
	storetypes "cosmossdk.io/store/types"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	rosettaCmd "cosmossdk.io/tools/rosetta/cmd"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	cosmosevmcmd "github.com/cosmos/evm/client"
	evmkeyring "github.com/cosmos/evm/crypto/keyring"
	evmserver "github.com/cosmos/evm/server"
	evmserverconfig "github.com/cosmos/evm/server/config"
	srvflags "github.com/cosmos/evm/server/flags"

	kiichain "github.com/kiichain/kiichain/v5/app"
)

// CustomAppConfig generates a new custom config
type CustomAppConfig struct {
	serverconfig.Config

	// EVM config
	EVM     evmserverconfig.EVMConfig
	JSONRPC evmserverconfig.JSONRPCConfig
	TLS     evmserverconfig.TLSConfig

	// wasm config
	Wasm wasmtypes.NodeConfig `mapstructure:"wasm"`
}

// NewRootCmd creates root command for kiichaind
func NewRootCmd() *cobra.Command {
	initAppOptions := viper.New()
	temp := createTempDir()
	initAppOptions.Set(flags.FlagHome, temp)

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
			_ = os.RemoveAll(temp) // cleanup temp dir
		}
	}()

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

			// Only enable SIGN_MODE_TEXTUAL if online
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

	initRootCmd(rootCmd, tempApp.ModuleBasics, tempApp.AppCodec(), tempApp.InterfaceRegistry(), tempApp.GetTxConfig())

	if err := enrichAutoCliOpts(tempApp.AutoCliOpts(), initClientCtx).EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd
}

func enrichAutoCliOpts(autoCliOpts autocli.AppOptions, clientCtx client.Context) autocli.AppOptions {
	autoCliOpts.AddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	autoCliOpts.ValidatorAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	autoCliOpts.ConsensusAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	autoCliOpts.ClientCtx = clientCtx

	return autoCliOpts
}

// initCometConfig helps to override default CometBFT Config values.
func initCometConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40
	return cfg
}

func initAppConfig(evmChainID uint64) (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.StateSync.SnapshotInterval = 1000
	srvCfg.StateSync.SnapshotKeepRecent = 10

	evmCfg := evmserverconfig.DefaultEVMConfig()
	evmCfg.EVMChainID = evmChainID

	customAppConfig := CustomAppConfig{
		Config:  *srvCfg,
		EVM:     *evmCfg,
		JSONRPC: *evmserverconfig.DefaultJSONRPCConfig(),
		TLS:     *evmserverconfig.DefaultTLSConfig(),
		Wasm:    wasmtypes.DefaultNodeConfig(),
	}

	defaultAppTemplate := serverconfig.DefaultConfigTemplate + wasmtypes.DefaultConfigTemplate()
	defaultAppTemplate += evmserverconfig.DefaultEVMConfigTemplate

	return defaultAppTemplate, customAppConfig
}

func initRootCmd(
	rootCmd *cobra.Command,
	basicManager module.BasicManager,
	cdc codec.Codec,
	interfaceRegistry codectypes.InterfaceRegistry,
	txConfig client.TxConfig,
) {
	cfg := sdk.GetConfig()
	cfg.Seal()

	ac := appCreator{}

	sdkAppCreatorWrapper := func(l log.Logger, d dbm.DB, w io.Writer, ao servertypes.AppOptions) servertypes.Application {
		return ac.newApp(l, d, w, ao)
	}
	rootCmd.AddCommand(
		initCmd(basicManager, kiichain.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(sdkAppCreatorWrapper, kiichain.DefaultNodeHome),
		snapshot.Cmd(sdkAppCreatorWrapper),
	)

	// EVM commands
	evmserver.AddCommands(
		rootCmd,
		evmserver.NewDefaultStartOptions(ac.newApp, kiichain.DefaultNodeHome),
		ac.appExport,
		addModuleInitFlags,
	)

	// Cosmos EVM key commands
	rootCmd.AddCommand(
		cosmosevmcmd.KeyCommands(kiichain.DefaultNodeHome, true),
	)

	// keybase, RPC, query, tx
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(basicManager),
		keys.Commands(),
	)

	// Rosetta
	rootCmd.AddCommand(rosettaCmd.RosettaCommand(interfaceRegistry, cdc))
}

func addModuleInitFlags(startCmd *cobra.Command) {
	wasm.AddModuleInitFlags(startCmd)
	overrideEVMChainID(startCmd)
}

func overrideEVMChainID(cmd *cobra.Command) {
	preCheck := func(cmd *cobra.Command, _ []string) error {
		evmChainID, err := cmd.Flags().GetUint64(srvflags.EVMChainID)
		if err != nil {
			return err
		}
		if evmChainID == evmserverconfig.DefaultEVMChainID {
			err = cmd.Flags().Set(srvflags.EVMChainID, fmt.Sprintf("%d", kiichain.KiichainID))
		}
		return err
	}

	cmd.PreRunE = chainPreRuns(preCheck, cmd.PreRunE)
}

type preRunFn func(cmd *cobra.Command, args []string) error

func chainPreRuns(pfns ...preRunFn) preRunFn {
	return func(cmd *cobra.Command, args []string) error {
		for _, pfn := range pfns {
			if pfn != nil {
				if err := pfn(cmd, args); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.GenesisCoreCommand(txConfig, basicManager, kiichain.DefaultNodeHome)
	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.ValidatorCommand(),
		server.QueryBlocksCmd(),
		server.QueryBlockCmd(),
		server.QueryBlockResultsCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")
	return cmd
}

func txCommand(basicManager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	basicManager.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

type appCreator struct{}

func (a appCreator) newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) evmserver.Application {
	var cache storetypes.MultiStorePersistentCache
	if cast.ToBool(appOpts.Get(server.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	var wasmOpts []wasmkeeper.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	chainID, err := getChainIDFromOpts(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotDir := filepath.Join(homeDir, "data", "snapshots")
	snapshotDB, err := dbm.NewDB("metadata", server.GetAppDBBackend(appOpts), snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get(server.FlagStateSyncSnapshotInterval)),
		cast.ToUint32(appOpts.Get(server.FlagStateSyncSnapshotKeepRecent)),
	)
	baseappOptions := []func(*baseapp.BaseApp){
		baseapp.SetChainID(chainID),
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(server.FlagMinGasPrices))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(server.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(server.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(server.FlagMinRetainBlocks))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(server.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(server.FlagIndexEvents))),
		baseapp.SetSnapshot(snapshotStore, snapshotOptions),
		baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get(server.FlagIAVLCacheSize))),
	}

	return kiichain.NewKiichainApp(
		logger,
		db,
		traceStore,
		true,
		skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		appOpts,
		wasmOpts,
		kiichain.EVMAppOptions,
		baseappOptions...,
	)
}

func (a appCreator) appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var kiichainApp *kiichain.KiichainApp

	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home is not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}
	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	var loadLatest bool
	if height == -1 {
		loadLatest = true
	}

	chainID, err := getChainIDFromOpts(appOpts)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	var emptyWasmOpts []wasmkeeper.Option
	kiichainApp = kiichain.NewKiichainApp(
		logger,
		db,
		traceStore,
		loadLatest,
		map[int64]bool{},
		homePath,
		appOpts,
		emptyWasmOpts,
		kiichain.EVMAppOptions,
		baseapp.SetChainID(chainID),
	)

	if height != -1 {
		if err := kiichainApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return kiichainApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

func createTempDir() string {
	dir, err := os.MkdirTemp("", ".kiichain")
	if err != nil {
		panic(fmt.Sprintf("failed creating temp directory: %s", err.Error()))
	}
	return dir
}

func getChainIDFromOpts(appOpts servertypes.AppOptions) (chainID string, err error) {
	chainID = cast.ToString(appOpts.Get(flags.FlagChainID))
	if chainID == "" {
		homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
		genDocFile := filepath.Join(homeDir, cast.ToString(appOpts.Get("genesis_file")))
		appGenesis, err := genutiltypes.AppGenesisFromFile(genDocFile)
		if err != nil {
			return "", err
		}
		chainID = appGenesis.ChainID
	}
	return
}

