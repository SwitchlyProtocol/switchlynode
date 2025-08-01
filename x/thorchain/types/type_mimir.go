package types

import (
	"strings"

	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

func (m NodeMimirs) Has(key string, acc cosmos.AccAddress) bool {
	for _, mim := range m.Mimirs {
		if mim.Signer.Equals(acc) && strings.EqualFold(mim.Key, key) {
			return true
		}
	}
	return false
}

func (m NodeMimirs) Get(key string, acc cosmos.AccAddress) (int64, bool) {
	for _, mim := range m.Mimirs {
		if mim.Signer.Equals(acc) && strings.EqualFold(mim.Key, key) {
			return mim.Value, true
		}
	}
	return 0, false
}

func (m *NodeMimirs) Set(key string, val int64, acc cosmos.AccAddress) {
	for i, mim := range m.Mimirs {
		if mim.Signer.Equals(acc) && strings.EqualFold(mim.Key, key) {
			m.Mimirs[i].Value = val
			return
		}
	}
	m.Mimirs = append(m.Mimirs, NodeMimir{
		Key:    key,
		Value:  val,
		Signer: acc,
	})
}

func (m *NodeMimirs) Delete(key string, acc cosmos.AccAddress) {
	for i, mim := range m.Mimirs {
		if mim.Signer.Equals(acc) && strings.EqualFold(mim.Key, key) {
			m.Mimirs = append(m.Mimirs[:i], m.Mimirs[i+1:]...)
			return
		}
	}
}

func (m NodeMimirs) countActive(key string, active []cosmos.AccAddress, maj func(_, _ int) bool) (int64, bool) {
	counter := make(map[int64]int) // count how many votes are for each value
	voted := make(map[string]bool) // track signers that have already voted
	for _, mimir := range m.Mimirs {
		// skip mismatching keys
		if !strings.EqualFold(mimir.Key, key) {
			continue
		}

		// skip signers we've already seend (no duplicates allowed)
		if v, ok := voted[mimir.Signer.String()]; v && ok {
			continue
		}

		for _, acc := range active {
			// skip if not the mimir's signer
			if !acc.Equals(mimir.Signer) {
				continue
			}

			voted[mimir.Signer.String()] = true // mark signer as voted
			if _, ok := counter[mimir.Value]; !ok {
				counter[mimir.Value] = 0
			}
			counter[mimir.Value]++
			break // Having confirmed the mimir's signer is active, go to the next mimir.
		}
	}

	// analyze-ignore(map-iteration)
	for val, count := range counter {
		if maj(count, len(active)) {
			return val, true
		}
	}

	return 0, false
}

func (m NodeMimirs) HasSuperMajority(key string, nas []cosmos.AccAddress) (int64, bool) {
	return m.countActive(key, nas, HasSuperMajority)
}

func (m NodeMimirs) HasSimpleMajority(key string, nas []cosmos.AccAddress) (int64, bool) {
	return m.countActive(key, nas, HasSimpleMajority)
}

func (m NodeMimirs) HasMinority(key string, nas []cosmos.AccAddress) (int64, bool) {
	// NOT IMPLEMENTED
	// Minotirty is a bit tricky, because a set can have multiple minorities, which can result in a potential consensus failure
	return 0, false
}

// ValueOfEconomic - fetches the value of a given mimir based on 2/3rds consensus
func (m NodeMimirs) ValueOfEconomic(key string, active []cosmos.AccAddress) int64 {
	voteCount := make(map[int64]int)
	hasVoted := make(map[string]bool)
	totalValidVotes := 0

	for _, mimir := range m.Mimirs {
		if !strings.EqualFold(mimir.Key, key) {
			continue
		}

		if hasVoted[mimir.Signer.String()] {
			continue // Skip this vote since the node already voted
		}

		// Ensure that the vote is only from active nodes
		for _, addr := range active {
			if addr.Equals(mimir.Signer) {
				voteCount[mimir.Value]++
				totalValidVotes++
				hasVoted[mimir.Signer.String()] = true
				break
			}
		}
	}

	mostVotedValue := int64(-1)
	maxVotes := 0
	// analyze-ignore(map-iteration)
	for value, count := range voteCount {
		if count > maxVotes {
			mostVotedValue = value
			maxVotes = count
		}
	}

	// Check if maxVotes is at least two-thirds of totalValidVotes using integer arithmetic
	if 3*maxVotes < 2*len(active) {
		return -1
	}

	return mostVotedValue
}

// ValueOfOperational - fetches the value of a given mimir based most votes (above min vote)
func (m NodeMimirs) ValueOfOperational(key string, minVotes int64, active []cosmos.AccAddress) int64 {
	voteCount := make(map[int64]int)
	hasVoted := make(map[string]bool)

	for _, mimir := range m.Mimirs {
		if !strings.EqualFold(mimir.Key, key) {
			continue
		}

		if hasVoted[mimir.Signer.String()] {
			continue // Skip this vote since the node already voted
		}

		// Ensure that the vote is only from active nodes
		for _, addr := range active {
			if addr.Equals(mimir.Signer) {
				voteCount[mimir.Value]++
				hasVoted[mimir.Signer.String()] = true
				break
			}
		}
	}

	mostVotedValue := int64(-1)
	maxVotes := 0
	tie := false
	// analyze-ignore(map-iteration)
	for value, count := range voteCount {
		if count > maxVotes {
			mostVotedValue = value
			maxVotes = count
		} else if count == maxVotes {
			tie = true
		}
	}

	if tie || int64(maxVotes) < minVotes {
		return -1
	}

	return mostVotedValue
}
