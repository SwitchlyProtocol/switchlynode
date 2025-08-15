package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
)

////////////////////////////////////////////////////////////////////////////////////////
// Run
////////////////////////////////////////////////////////////////////////////////////////

func run(out io.Writer, path string, routine int, doneWithRetries func(path string) bool, abort <-chan struct{}) (failExportInvariants bool, err error) {
	localLog := consoleLogger(out)

	home := "/" + strconv.Itoa(routine)
	localLog.Info().Str("path", path).Msgf("Loading regression test")

	// clear data directory
	localLog.Debug().Msg("Clearing data directory")
	switchlynodePath := filepath.Join(home, ".switchlynode")
	cmdOut, err := exec.Command("rm", "-rf", switchlynodePath).CombinedOutput()
	if err != nil {
		fmt.Println(string(cmdOut))
		log.Fatal().Err(err).Msg("failed to clear data directory")
	}

	// use same environment for all commands
	env := []string{
		"HOME=" + home,
		"SIGNER_NAME=switchly",
		"SIGNER_PASSWD=password",
		"CHAIN_HOME_FOLDER=" + switchlynodePath,
		"SWITCHLY_TENDERMINT_INSTRUMENTATION_PROMETHEUS=false",
		// block time should be short, but all consecutive checks must complete within timeout
		fmt.Sprintf("SWITCHLY_TENDERMINT_CONSENSUS_TIMEOUT_COMMIT=%s", time.Second*getTimeFactor()),
		// all ports will be offset by the routine number
		fmt.Sprintf("SWITCHLY_COSMOS_API_ADDRESS=tcp://0.0.0.0:%d", 1317+routine),
		fmt.Sprintf("SWITCHLY_COSMOS_EBIFROST_ADDRESS=127.0.0.1:%d", 50051+routine),
		fmt.Sprintf("SWITCHLY_TENDERMINT_RPC_LISTEN_ADDRESS=tcp://0.0.0.0:%d", 26657+routine),
		fmt.Sprintf("SWITCHLY_TENDERMINT_P2P_LISTEN_ADDRESS=tcp://0.0.0.0:%d", 27000+routine),
		"CREATE_BLOCK_PORT=" + strconv.Itoa(8080+routine),
		"GOCOVERDIR=/mnt/coverage",
	}

	// if DEBUG is set also output switchlynode debug logs
	debugVar := os.Getenv("DEBUG")
	if debugVar != "" {
		env = append(env, "SWITCHLY_TENDERMINT_LOG_LEVEL="+debugVar)
	}

	// init chain with dog mnemonic
	localLog.Debug().Msg("Initializing chain")
	cmd := exec.Command("switchlynode", "init", "local", "--chain-id", "switchly", "--recover")
	cmd.Stdin = bytes.NewBufferString(dogMnemonic + "\n")
	cmd.Env = env
	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(cmdOut))
		log.Fatal().Err(err).Msg("failed to initialize chain")
	}

	// init keys with dog mnemonic
	localLog.Debug().Msg("Initializing keys")
	cmd = exec.Command("switchlynode", "keys", "--keyring-backend=file", "add", "--recover", "switchly")
	cmd.Stdin = bytes.NewBufferString(dogMnemonic + "\npassword\npassword\n")
	cmd.Env = env
	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(cmdOut))
		log.Fatal().Err(err).Msg("failed to initialize keys")
	}

	// init chain
	localLog.Debug().Msg("Initializing chain")
	cmd = exec.Command("switchlynode", "init", "local", "--chain-id", "switchly", "-o")
	cmd.Env = env
	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(cmdOut))
		log.Fatal().Err(err).Msg("failed to initialize chain")
	}

	// create routine local state (used later by custom template functions in operations)
	nativeTxIDsMu.Lock()
	nativeTxIDs[routine] = []string{}
	nativeTxIDsMu.Unlock()
	tmpls := template.Must(templates.Clone())

	// ensure no naming collisions
	if tmpls.Lookup(filepath.Base(path)) != nil {
		log.Fatal().Msgf("test name collision: %s", filepath.Base(path))
	}

	ops, opLines, env, failExportInvariants := parseOps(localLog, path, tmpls, env)

	// warn if no operations found
	if len(ops) == 0 {
		err = errors.New("no operations found")
		localLog.Err(err).Msg("")
		return false, err
	}

	localLog.Info().Str("path", path).Int("blocks", blockCount(ops)).Msgf("Running regression test")

	// clear block directory
	blocksPath := filepath.Join("/mnt/blocks", strings.TrimPrefix(path, "suites/"))
	blocksPath = strings.TrimSuffix(blocksPath, ".yaml")
	_ = os.RemoveAll(blocksPath)

	// execute all state operations
	stateOpCount := 0
	for i, op := range ops {
		select {
		case <-abort:
			return false, nil
		default:
		}
		if _, ok := op.(*OpState); ok {
			localLog.Info().Int("line", opLines[i]).Msgf(">>> [%d] %s", i+1, op.OpType())
			err = op.Execute(out, path, opLines[i], routine, cmd.Process, nil)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to execute state operation")
			}
			stateOpCount++
		}
	}
	ops = ops[stateOpCount:]
	opLines = opLines[stateOpCount:]

	// validate genesis
	localLog.Debug().Msg("Validating genesis")
	cmd = exec.Command("switchlynode", "genesis", "validate")
	cmd.Env = env
	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		cmdErr := err
		// dump the genesis
		fmt.Println(ColorPurple + "Genesis:" + ColorReset)
		var f *os.File
		f, err = os.OpenFile(filepath.Join(home, ".switchlynode/config/genesis.json"), os.O_RDWR, 0o644)
		if err != nil {
			log.Fatal().Err(err).Str("path", path).Msg("failed to open genesis file")
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		f.Close()

		// dump error and exit
		fmt.Println(string(cmdOut))
		log.Fatal().Err(cmdErr).Msg("genesis validation failed")
	}

	// render config
	localLog.Debug().Msg("Rendering config")
	cmd = exec.Command("switchlynode", "render-config")
	cmd.Env = env
	err = cmd.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to render config")
	}

	// overwrite private validator key
	localLog.Debug().Msg("Overwriting private validator key")
	keyPath := filepath.Join(home, ".switchlynode/config/priv_validator_key.json")
	cmd = exec.Command("cp", "/mnt/priv_validator_key.json", keyPath)
	err = cmd.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to overwrite private validator key")
	}

	logLevel := "info"
	switch debugVar {
	case "trace", "debug", "info", "warn", "error", "fatal", "panic":
		logLevel = debugVar
	}

	// setup process io
	switchlynode := exec.Command("/regtest/cover-switchlynode", "--log_level", logLevel, "start")
	switchlynode.Env = env

	stderr, err := switchlynode.StderrPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup switchlynode stderr")
	}
	stderrScanner := bufio.NewScanner(stderr)
	stderrLines := make(chan string, 100)
	go func() {
		for stderrScanner.Scan() {
			stderrLines <- stderrScanner.Text()
		}
	}()
	if debugVar != "" {
		switchlynode.Stdout = os.Stdout
		switchlynode.Stderr = os.Stderr
	}

	// start switchlynode process
	localLog.Debug().Msg("Starting switchlynode")
	err = switchlynode.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start switchlynode")
	}

	// wait for switchlynode to listen on block creation port
	time.Sleep(time.Second)
	for i := 0; ; i++ {
		if i%100 == 0 {
			localLog.Debug().Msg("Waiting for switchlynode to listen")
		}
		time.Sleep(100 * time.Millisecond)
		var conn net.Conn
		conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", 8080+routine))
		if err == nil {
			conn.Close()
			break
		}
	}

	// run the operations
	var returnErr error
	localLog.Info().Msgf("Executing %d operations", len(ops))
	for i, op := range ops {
		select {
		case <-abort:
			return false, nil
		default:
		}

		// prefetch sequences of check operations
		if os.Getenv("AUTO_UPDATE") == "" && op.OpType() == "check" && ops[i-1].OpType() != "check" {
			wg := sync.WaitGroup{}
			for j := i; j < len(ops); j++ {
				select {
				case <-abort:
					return false, nil
				default:
				}
				if ops[j].OpType() != "check" {
					break
				}
				wg.Add(1)
				go func(j int) {
					defer wg.Done()
					// trunk-ignore(golangci-lint/forcetypeassert)
					ops[j].(*OpCheck).prefetch(routine)
				}(j)
			}
			wg.Wait()
		}

		localLog.Info().Int("line", opLines[i]).Msgf(">>> [%d] %s", stateOpCount+i+1, op.OpType())
		returnErr = op.Execute(out, path, opLines[i], routine, switchlynode.Process, stderrLines)
		if returnErr != nil {
			localLog.Error().Err(returnErr).
				Int("line", opLines[i]).
				Int("op", stateOpCount+i+1).
				Str("type", op.OpType()).
				Str("path", path).
				Msg("operation failed")
			dumpLogs(out, stderrLines)
			break
		}

		// if in AUTO_UPDATE and the operation was a check, re-render future operations
		if os.Getenv("AUTO_UPDATE") != "" && op.OpType() == "check" {
			newOps, _, _, _ := parseOps(localLog, path, tmpls, env)
			remainingOps := len(ops) - i - 1
			newOps = newOps[len(newOps)-remainingOps:]
			for j, newOp := range newOps {
				ops[i+j+1] = newOp
			}
		}
	}

	// log success
	if returnErr == nil {
		localLog.Info().Msg("All operations succeeded")
	}

	// stop switchlynode process
	localLog.Debug().Msg("Stopping switchlynode")
	err = switchlynode.Process.Signal(syscall.SIGUSR1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to stop switchlynode")
	}

	// wait for process to exit
	_, err = switchlynode.Process.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to wait for switchlynode")
	}

	// retry context deadline exceeded errors and all errors in CI
	if returnErr != nil && (strings.Contains(returnErr.Error(), "context deadline exceeded") || os.Getenv("CI") != "") {
		log.Warn().Err(returnErr).Str("path", path).Msg("retrying suite")
		fmt.Println()

		if doneWithRetries(path) {
			return false, returnErr
		}

		return run(out, path, routine, doneWithRetries, abort)
	}

	// if failed and debug enabled restart to allow inspection
	if returnErr != nil && os.Getenv("DEBUG") != "" {

		// remove validator key (otherwise switchlynode will hang in begin block)
		localLog.Debug().Msg("Removing validator key")
		cmd = exec.Command("rm", keyPath)
		cmdOut, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(cmdOut))
			log.Fatal().Err(err).Msg("failed to remove validator key")
		}

		// restart switchlynode
		localLog.Debug().Msg("Restarting switchlynode")
		switchlynode = exec.Command("switchlynode", "--log_level", logLevel, "start")
		switchlynode.Env = env
		switchlynode.Stdout = os.Stdout
		switchlynode.Stderr = os.Stderr
		err = switchlynode.Start()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to restart switchlynode")
		}

		// wait for switchlynode
		localLog.Debug().Msg("Waiting for switchlynode")
		_, err = switchlynode.Process.Wait()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to wait for switchlynode")
		}
	}

	return failExportInvariants, returnErr
}
