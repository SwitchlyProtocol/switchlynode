// Package dag provides execution of an actor DAG (directed acyclic graph).
package dag

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	. "github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/types"
)

// Execute executes the actor DAG from the provided root. It is precondition that the
// root actor points to a proper DAG and contains no cycles.
func Execute(c *OpConfig, root *Actor, parallelism int) {
	// determine the total number of actors in dag
	seen := map[*Actor]bool{}
	root.WalkDepthFirst(func(a *Actor) bool {
		seen[a] = true
		return true
	})
	total := len(seen)

	// initialize dag
	root.InitRoot()
	sem := make(chan struct{}, parallelism)

	status := func() (ready, running, finished map[*Actor]bool) {
		// determine all actors that are ready to execute
		ready = map[*Actor]bool{}
		finished = map[*Actor]bool{}
		running = map[*Actor]bool{}
		root.WalkDepthFirst(func(a *Actor) bool {
			if a.Finished() {
				finished[a] = true
				return true
			}
			if a.Started() || a.Backgrounded() {
				running[a] = true
				return true
			}

			// all parents must be finished or backgrounded to start
			for parent := range a.Parents() {
				if !parent.Finished() && !parent.Backgrounded() {
					return false
				}
			}

			ready[a] = true
			return true
		})

		return ready, running, finished
	}

	// execute dag
	log.Info().Int("actors", total).Int("parallelism", parallelism).Msg("executing dag")
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for range tick.C {
		ready, running, finished := status()

		// if all actors are finished we are done
		if len(finished) == total {
			log.Info().Int("actors", len(finished)).Msg("simulation finished successfully")
			return
		}

		// info log context
		infoLog := log.Info().
			Int("finished", len(finished)).
			Int("running", len(running)).
			Int("remaining", total-len(finished)).
			Int("ready", len(ready))

		// determine how many users are available
		lockedUsers := 0
		availableUsers := 0
		for _, user := range c.Users {
			if !user.IsLocked() {
				availableUsers++
			} else {
				lockedUsers++
			}
		}

		// sleep if no actors are ready to execute
		if len(ready) == 0 || availableUsers == 0 {
			infoLog.
				Int("available_users", availableUsers).
				Int("locked_users", lockedUsers).
				Msg("waiting for ready actors and users")

			continue
		}

		// randomly select an actor to execute
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(ready))))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to generate random number")
		}
		readySlice := make([]*Actor, 0, len(ready))
		for a := range ready {
			readySlice = append(readySlice, a)
		}
		a := readySlice[random.Int64()]

		// execute actor
		infoLog.Str("actor", a.Name).Msg("executing actor")
		a.Start()
		sem <- struct{}{}
		go func(a *Actor, start time.Time) {
			defer func() {
				duration := time.Since(start) / time.Second * time.Second // round to second
				a.Log().Info().Str("duration", duration.String()).Msg("finished")
				<-sem
			}()

			// tee the actor logs to a buffer that we dump if it fails
			buf := new(bytes.Buffer)
			teeWriter := zerolog.MultiLevelWriter(buf, os.Stdout)
			a.SetLogger(a.Log().Output(zerolog.ConsoleWriter{
				Out:        teeWriter,
				TimeFormat: time.TimeOnly,
			}))

			err = a.Execute(c)
			if err != nil {
				os.Stderr.Write([]byte("\n\nFailed actor logs:\n" + buf.String() + "\n\n"))

				// print currently running actors
				os.Stderr.Write([]byte("\n\nCurrently running actors:\n"))
				running, _, _ := status()
				for a := range running {
					os.Stderr.Write([]byte(a.Name + "\n"))
				}
				os.Stderr.Write([]byte("\n\n"))

				a.Log().Fatal().Err(err).Msg("actor execution failed")
			}
		}(a, time.Now())
	}
}
