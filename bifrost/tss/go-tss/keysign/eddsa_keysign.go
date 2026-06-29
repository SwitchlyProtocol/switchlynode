package keysign

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"time"

	tsslibcommon "github.com/binance-chain/tss-lib/common"
	eddsakg "github.com/binance-chain/tss-lib/eddsa/keygen"
	eddsasigning "github.com/binance-chain/tss-lib/eddsa/signing"
	btss "github.com/binance-chain/tss-lib/tss"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/messages"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/storage"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/blame"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/common"
)

// SignMessageEdDSA runs an EdDSA (ed25519) threshold signing ceremony. It mirrors the secp256k1
// SignMessage but uses tss-lib's eddsa/signing party with the keyshare from
// KeygenLocalState.EdDSALocalData. The caller MUST have set the tss-lib global curve to Edwards (the
// go-tss server does not touch the global curve per-ceremony — see docs/architecture/
// stellar-eddsa-tss.md §9). The ECDSA SignMessage in tss_keysign.go is left byte-for-byte unchanged.
func (tKeySign *TssKeySign) SignMessageEdDSA(msgsToSign [][]byte, localStateItem storage.KeygenLocalState, parties []string) ([]*tsslibcommon.ECSignature, error) {
	partiesID, localPartyID, err := conversion.GetParties(parties, localStateItem.LocalPartyKey)
	if err != nil {
		return nil, fmt.Errorf("fail to form key sign party: %w", err)
	}
	if !common.Contains(partiesID, localPartyID) {
		tKeySign.logger.Info().Msgf("we are not in this rounds key sign")
		return nil, nil
	}
	if len(localStateItem.EdDSALocalData) == 0 {
		return nil, errors.New("no eddsa keyshare in local state")
	}
	var keyData eddsakg.LocalPartySaveData
	if err := json.Unmarshal(localStateItem.EdDSALocalData, &keyData); err != nil {
		return nil, fmt.Errorf("fail to unmarshal eddsa keyshare: %w", err)
	}
	threshold, err := conversion.GetThreshold(len(localStateItem.ParticipantKeys))
	if err != nil {
		return nil, errors.New("fail to get threshold")
	}

	outCh := make(chan btss.Message, 2*len(partiesID)*len(msgsToSign))
	endCh := make(chan *eddsasigning.SignatureData, len(partiesID)*len(msgsToSign))
	errCh := make(chan struct{})

	keySignPartyMap := new(sync.Map)
	for i, val := range msgsToSign {
		m, err := common.MsgToHashInt(val)
		if err != nil {
			return nil, fmt.Errorf("fail to convert msg to hash int: %w", err)
		}
		moniker := m.String() + ":" + strconv.Itoa(i)
		partiesID, eachLocalPartyID, err := conversion.GetParties(parties, localStateItem.LocalPartyKey)
		if err != nil {
			return nil, fmt.Errorf("error to create parties in batch signing %w", err)
		}
		ctx := btss.NewPeerContext(partiesID)
		eachLocalPartyID.Moniker = moniker
		tKeySign.localParties = nil
		params := btss.NewParameters(ctx, eachLocalPartyID, len(partiesID), threshold)
		keySignParty := eddsasigning.NewLocalParty(m, params, keyData, outCh, endCh)
		keySignPartyMap.Store(moniker, keySignParty)
	}

	blameMgr := tKeySign.tssCommonStruct.GetBlameMgr()
	partyIDMap := conversion.SetupPartyIDMap(partiesID)
	err1 := conversion.SetupIDMaps(partyIDMap, tKeySign.tssCommonStruct.PartyIDtoP2PID)
	err2 := conversion.SetupIDMaps(partyIDMap, blameMgr.PartyIDtoP2PID)
	if err1 != nil || err2 != nil {
		tKeySign.logger.Error().Err(err).Msgf("error in creating mapping between partyID and P2P ID")
		return nil, err
	}

	tKeySign.tssCommonStruct.SetPartyInfo(&common.PartyInfo{
		PartyMap:   keySignPartyMap,
		PartyIDMap: partyIDMap,
	})
	blameMgr.SetPartyInfo(keySignPartyMap, partyIDMap)
	tKeySign.tssCommonStruct.P2PPeersLock.Lock()
	tKeySign.tssCommonStruct.P2PPeers = conversion.GetPeersID(tKeySign.tssCommonStruct.PartyIDtoP2PID, tKeySign.tssCommonStruct.GetLocalPeerID())
	tKeySign.tssCommonStruct.P2PPeersLock.Unlock()
	var keySignWg sync.WaitGroup
	keySignWg.Add(2)
	go func() {
		defer keySignWg.Done()
		// startBatchSigning is generic over btss.Party, so it is reused for EdDSA.
		ret := tKeySign.startBatchSigning(keySignPartyMap, len(msgsToSign))
		if !ret {
			close(errCh)
		}
	}()
	go tKeySign.tssCommonStruct.ProcessInboundMessages(tKeySign.commStopChan, &keySignWg)
	results, err := tKeySign.processKeySignEdDSA(len(msgsToSign), errCh, outCh, endCh)
	if err != nil {
		close(tKeySign.commStopChan)
		return nil, fmt.Errorf("fail to process eddsa key sign: %w", err)
	}

	select {
	case <-time.After(time.Second * 5):
		close(tKeySign.commStopChan)
	case <-tKeySign.tssCommonStruct.GetTaskDone():
		close(tKeySign.commStopChan)
	}
	keySignWg.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		a := new(big.Int).SetBytes(results[i].M)
		b := new(big.Int).SetBytes(results[j].M)
		return a.Cmp(b) != -1
	})
	return results, nil
}

func (tKeySign *TssKeySign) processKeySignEdDSA(reqNum int, errChan chan struct{}, outCh <-chan btss.Message, endCh <-chan *eddsasigning.SignatureData) ([]*tsslibcommon.ECSignature, error) {
	defer tKeySign.logger.Debug().Msg("eddsa key sign finished")
	var signatures []*tsslibcommon.ECSignature
	tssConf := tKeySign.tssCommonStruct.GetConf()
	blameMgr := tKeySign.tssCommonStruct.GetBlameMgr()

	for {
		select {
		case <-errChan:
			tKeySign.logger.Error().Msg("eddsa key sign failed")
			return nil, errors.New("error channel closed fail to start local party")
		case <-tKeySign.stopChan:
			return nil, errors.New("received exit signal")
		case <-time.After(tssConf.KeySignTimeout):
			tKeySign.keysignTimeoutBlame()
			return nil, blame.ErrTssTimeOut
		case msg := <-outCh:
			tKeySign.logger.Debug().Msgf(">>>>>>>>>>eddsa key sign msg: %s", msg.String())
			blameMgr.SetLastMsg(msg)
			blameMgr.GetBlame().Round = msg.Type()
			if err := tKeySign.tssCommonStruct.ProcessOutCh(msg, messages.TSSKeySignMsg); err != nil {
				return nil, err
			}
		case msg := <-endCh:
			signatures = append(signatures, msg.GetSignature())
			if len(signatures) == reqNum {
				tKeySign.logger.Debug().Msg("we have done the eddsa key sign")
				if err := tKeySign.tssCommonStruct.NotifyTaskDone(); err != nil {
					tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
				}
				address := tKeySign.p2pComm.ExportPeerAddress()
				if err := tKeySign.stateManager.SaveAddressBook(address); err != nil {
					tKeySign.logger.Error().Err(err).Msg("fail to save the peer addresses")
				}
				return signatures, nil
			}
		}
	}
}

// keysignTimeoutBlame replicates the blame analysis performed on a keysign timeout. The ECDSA
// processKeySign keeps its own inline copy (so tss_keysign.go is untouched); the EdDSA path uses this
// shared helper rather than duplicate it again.
func (tKeySign *TssKeySign) keysignTimeoutBlame() {
	tssConf := tKeySign.tssCommonStruct.GetConf()
	blameMgr := tKeySign.tssCommonStruct.GetBlameMgr()
	tKeySign.logger.Error().Msgf("fail to sign message with %s", tssConf.KeySignTimeout.String())
	lastMsg := blameMgr.GetLastMsg()
	failReason := blameMgr.GetBlame().FailReason
	if failReason == "" {
		failReason = blame.TssTimeout
	}
	tKeySign.tssCommonStruct.P2PPeersLock.RLock()
	threshold, err := conversion.GetThreshold(len(tKeySign.tssCommonStruct.P2PPeers) + 1)
	tKeySign.tssCommonStruct.P2PPeersLock.RUnlock()
	if err != nil {
		tKeySign.logger.Error().Err(err).Msg("error in get the threshold for generate blame")
	}
	if lastMsg != nil && !lastMsg.IsBroadcast() {
		blameNodesUnicast, err := blameMgr.GetUnicastBlame(lastMsg.Type())
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("error in get unicast blame")
		}
		if len(blameNodesUnicast) > 0 && len(blameNodesUnicast) <= threshold {
			blameMgr.GetBlame().SetBlame(failReason, blameNodesUnicast, true, tKeySign.tssCommonStruct.RoundInfo)
		}
	} else if lastMsg != nil {
		blameNodesUnicast, err := blameMgr.GetUnicastBlame(conversion.GetPreviousKeySignUicast(lastMsg.Type()))
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("error in get unicast blame")
		}
		if len(blameNodesUnicast) > 0 && len(blameNodesUnicast) <= threshold {
			blameMgr.GetBlame().SetBlame(failReason, blameNodesUnicast, true, tKeySign.tssCommonStruct.RoundInfo)
		}
	}
	if lastMsg != nil {
		blameNodesBroadcast, err := blameMgr.GetBroadcastBlame(lastMsg.Type())
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("error in get broadcast blame")
		}
		blameMgr.GetBlame().AddBlameNodes(blameNodesBroadcast...)
	}
	if len(blameMgr.GetBlame().BlameNodes) == 0 {
		blameNodesMisingShare, isUnicast, err := blameMgr.TssMissingShareBlame(messages.TSSKEYSIGNROUNDS)
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("fail to get the node of missing share ")
		}
		if len(blameNodesMisingShare) > 0 && len(blameNodesMisingShare) <= threshold {
			blameMgr.GetBlame().AddBlameNodes(blameNodesMisingShare...)
			blameMgr.GetBlame().IsUnicast = isUnicast
		}
	}
}
