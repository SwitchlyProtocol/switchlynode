package watchers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	keeperv1 "github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper/v1"

	. "github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/types"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	},
	Timeout: 5 * time.Second,
}

func NewInvariants() *Watcher {
	cl := log.With().Str("watcher", "invariants").Logger()

	// gather the list of all invariants to watch
	invariants := []string{}
	k := keeperv1.KVStore{}
	for _, ir := range k.InvariantRoutes() {
		invariants = append(invariants, ir.Route)
	}

	return &Watcher{
		Name:     "Invariants",
		Interval: 10 * time.Second,
		Fn: func(config *OpConfig) error {
			for _, invariant := range invariants {
				endpoint := fmt.Sprintf("%s/thorchain/invariant/%s", thornodeURL, invariant)

				resp, err := httpClient.Get(endpoint)
				if err != nil {
					cl.Error().Err(err).Str("invariant", invariant).Msg("failed to get invariant")
					continue
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					cl.Error().
						Str("invariant", invariant).
						Int("status", resp.StatusCode).
						Msg("invariant returned non-200 status")
					continue
				}
				invRes := struct {
					Broken    bool
					Invariant string
					Msg       []string
				}{}
				if err = json.NewDecoder(resp.Body).Decode(&invRes); err != nil {
					cl.Error().Err(err).
						Str("invariant", invariant).
						Msg("failed to decode invariant response")
					continue
				}
				if invRes.Broken {
					msg := strings.Join(invRes.Msg, ", ")
					err = fmt.Errorf("invariant %s is broken: %s", invRes.Invariant, msg)
					cl.Error().Err(err).Msg("invariant is broken")
					return err
				}
			}
			return nil
		},
	}
}
