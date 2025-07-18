package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog/log"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	openapi "github.com/switchlyprotocol/switchlynode/v1/openapi/gen"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain"
)

////////////////////////////////////////////////////////////////////////////////////////
// Export
////////////////////////////////////////////////////////////////////////////////////////

func export(out io.Writer, path string, routine int, failExportInvariants bool) error {
	localLog := consoleLogger(out)
	home := "/" + strconv.Itoa(routine)

	// export blocks
	err := exportBlocks(out, path)
	if err != nil {
		return err
	}

	// export state
	localLog.Debug().Msg("Exporting state")
	cmd := exec.Command("thornode", "export", "--log_level=disabled")
	cmd.Env = append(os.Environ(), "HOME="+home)
	exportOut, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(exportOut))
		log.Fatal().Err(err).Msg("failed to export state")
	}

	// decode export
	var export map[string]any
	err = json.Unmarshal(exportOut, &export)
	if err != nil {
		fmt.Println(string(exportOut))
		log.Fatal().Err(err).Msg("failed to decode export")
	}

	// ignore genesis time and version for comparison
	delete(export, "app_version")
	delete(export, "genesis_time")
	appState, _ := export["app_state"].(map[string]any)
	thorchain, _ := appState["thorchain"].(map[string]any)
	delete(thorchain, "store_version")

	// ignore node account version for comparison
	nodeAccounts, _ := thorchain["node_accounts"].([]interface{})
	for i, na := range nodeAccounts {
		na, _ := na.(map[string]interface{})
		delete(na, "version")
		nodeAccounts[i] = na
	}

	// encode export
	exportOut, err = json.MarshalIndent(export, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to encode export")
	}

	// base path without extension and replace path separators with underscores
	path = strings.TrimPrefix(path, "suites/")
	exportName := strings.TrimSuffix(path, filepath.Ext(path))
	exportPath := fmt.Sprintf("/mnt/exports/%s.json", exportName)

	// check whether existing export exists
	_, err = os.Stat(exportPath)
	exportExists := err == nil

	// check export invariants
	err = checkExportInvariants(out, export)
	if err != nil && !failExportInvariants {
		// also log export changes for easier debugging
		if exportExists {
			_ = checkExportChanges(out, export, exportPath)
		}

		return err
	}

	// export if it none exists or EXPORT is set
	if !exportExists || os.Getenv("EXPORT") != "" {
		localLog.Debug().Msg("Writing export")

		// create the parent directory
		err = os.MkdirAll(filepath.Dir(exportPath), 0o700)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create export directory")
		}

		// write the export file
		err = os.WriteFile(exportPath, exportOut, 0o600)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write export")
		}
		return nil
	}

	return checkExportChanges(out, export, exportPath)
}

// exportBlocks will concatenate the blocks exports written during the create-blocks
// operations to a single file for the suite.
func exportBlocks(out io.Writer, path string) error {
	localLog := consoleLogger(out)

	// determine path for the export
	path = strings.TrimPrefix(path, "suites/")
	path = strings.TrimSuffix(path, filepath.Ext(path))
	blocksPath := fmt.Sprintf("/mnt/blocks/%s.json", path)
	blocksDir := strings.TrimSuffix(blocksPath, filepath.Ext(blocksPath))

	// read all block exports in the blocks directory
	files, err := os.ReadDir(blocksDir)
	if err != nil {
		localLog.Fatal().Err(err).Msg("failed to read block exports")
	}
	blocks := []openapi.BlockResponse{}
	for _, file := range files {
		// append all blocks to the export
		block := openapi.BlockResponse{}
		var f *os.File
		f, err = os.Open(filepath.Join(blocksDir, file.Name()))
		if err != nil {
			localLog.Fatal().Err(err).Msg("failed to open block export")
		}
		err = json.NewDecoder(f).Decode(&block)
		if err != nil {
			localLog.Fatal().Err(err).Msg("failed to decode block export")
		}
		blocks = append(blocks, block)
		f.Close()
	}

	// check whether existing blocks export exists
	_, err = os.Stat(blocksPath)
	exportExists := err == nil

	if !exportExists || os.Getenv("EXPORT") != "" {
		localLog.Debug().Msg("Writing blocks export")

		// create the parent directory
		err = os.MkdirAll(filepath.Dir(blocksPath), 0o700)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create blocks export directory")
		}

		// write the blocks file
		var f *os.File
		f, err = os.Create(blocksPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create blocks export")
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		err = enc.Encode(blocks)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write blocks export")
		}
		f.Close()

		return nil
	}

	return checkBlocksChanges(out, blocks, blocksPath)
}

////////////////////////////////////////////////////////////////////////////////////////
// Checks
////////////////////////////////////////////////////////////////////////////////////////

func checkExportInvariants(out io.Writer, genesis map[string]any) error {
	localLog := consoleLogger(out)

	// check export invariants
	localLog.Debug().Msg("Checking export invariants")
	appState, _ := genesis["app_state"].(map[string]any)

	// encode thorchain state to json for custom unmarshal
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(appState["thorchain"])
	if err != nil {
		log.Fatal().Err(err).Msg("failed to encode genesis state")
	}

	// unmarshal json to genesis state
	genesisState := &thorchain.GenesisState{}
	err = encodingConfig.Codec.UnmarshalJSON(buf.Bytes(), genesisState)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to decode genesis state")
	}

	// sum of pool + outbounds should be less than or equal to sum of vaults
	var sumPoolAsset, sumVaultAsset common.Coins
	poolAssets := map[common.Asset]bool{}
	for _, pool := range genesisState.Pools {
		poolAssets[pool.Asset] = true
		sumPoolAsset = sumPoolAsset.Add(common.NewCoin(pool.Asset, pool.BalanceAsset))
	}
	for _, vault := range genesisState.Vaults {
		// only count coins with pools
		for _, coin := range vault.Coins {
			if poolAssets[coin.Asset] {
				sumVaultAsset = sumVaultAsset.Add(coin)
			}
		}
	}
	for _, txout := range genesisState.TxOuts {
		for _, toi := range txout.TxArray {
			sumPoolAsset = sumPoolAsset.Add(toi.Coin)
		}
	}
	for _, tu := range genesisState.TradeUnits {
		coin := common.NewCoin(tu.Asset.GetLayer1Asset(), tu.Depth)
		sumPoolAsset = sumPoolAsset.Add(coin)
	}

	for _, p := range genesisState.SecuredAssets {
		coin := common.NewCoin(p.Asset.GetLayer1Asset(), p.Depth)
		sumPoolAsset = sumPoolAsset.Add(coin)
	}

	// adjust vault depths to ignore streaming swap amounts
	for _, swp := range genesisState.StreamingSwaps {
		for _, msg := range genesisState.SwapQueueItems {
			if swp.TxID.Equals(msg.Tx.ID) {
				sumVaultAsset = sumVaultAsset.SafeSub(common.NewCoin(msg.Tx.Coins[0].Asset, common.SafeSub(swp.Deposit, swp.In)))
				sumVaultAsset = sumVaultAsset.SafeSub(common.NewCoin(msg.TargetAsset, swp.Out))
			}
		}
	}

	// print any discrepancies
	for _, coin := range sumPoolAsset {
		for _, vaultCoin := range sumVaultAsset {
			if coin.Asset.Equals(vaultCoin.Asset) && !coin.Amount.Equal(vaultCoin.Amount) {
				if coin.Amount.GT(vaultCoin.Amount) {
					localLog.Error().Msgf("%s pool has %s more than its vaults\n", coin.Asset, common.SafeSub(coin.Amount, vaultCoin.Amount))
				}
				if vaultCoin.Amount.GT(coin.Amount) {
					localLog.Error().Msgf("%s vaults have %s more than their pool\n", coin.Asset, common.SafeSub(vaultCoin.Amount, coin.Amount))
				}
				err = errors.New("pool discrepancy")
			}
		}
	}

	// print outbounds for debugging
	if err != nil {
		for _, txout := range genesisState.TxOuts {
			for _, toi := range txout.TxArray {
				localLog.Error().Msgf("%s outbound: %s\n", toi.Coin.Asset, toi.Coin.Amount)
			}
		}
	}

	return err
}

func checkExportChanges(out io.Writer, newExport map[string]any, path string) error {
	localLog := consoleLogger(out)

	// compare existing export
	localLog.Debug().Msg("Reading existing export")

	// open existing export
	f, err := os.Open(path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open existing export")
	}
	defer f.Close()

	// decode existing export
	oldExport := map[string]any{}
	err = json.NewDecoder(f).Decode(&oldExport)
	if err != nil {
		localLog.Err(err).Msg("failed to decode existing export")
	}

	// compare exports
	log.Debug().Msg("Comparing exports")
	diff := cmp.Diff(oldExport, newExport)
	if diff != "" {
		localLog.Error().Msgf("exports differ: %s", diff)
		return errors.New("exports differ")
	}

	localLog.Info().Msg("State export matches expected")
	return nil
}

func checkBlocksChanges(out io.Writer, newBlocks []openapi.BlockResponse, path string) error {
	localLog := consoleLogger(out)

	// compare existing blocks
	localLog.Debug().Msg("Reading existing blocks")

	// open existing blocks
	f, err := os.Open(path)
	if err != nil {
		localLog.Fatal().Err(err).Msg("failed to open existing blocks")
	}
	defer f.Close()

	// decode existing blocks
	oldBlocks := []openapi.BlockResponse{}
	err = json.NewDecoder(f).Decode(&oldBlocks)
	if err != nil {
		localLog.Err(err).Msg("failed to decode existing blocks")
	}

	// compare blocks
	diff := cmp.Diff(oldBlocks, newBlocks)
	if diff != "" {
		localLog.Error().Msgf("blocks differ: %s", diff)
		return errors.New("blocks differ")
	}

	localLog.Info().Msg("Blocks export matches expected")
	return nil
}
