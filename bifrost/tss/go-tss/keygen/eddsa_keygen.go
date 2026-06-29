package keygen

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	bcrypto "github.com/binance-chain/tss-lib/crypto"
	eddsakg "github.com/binance-chain/tss-lib/eddsa/keygen"
	btss "github.com/binance-chain/tss-lib/tss"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/messages"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/storage"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/blame"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/common"
)

// keygenTimeoutBlame replicates the blame analysis performed on a keygen timeout. The ECDSA
// processKeyGen keeps its own inline copy (so tss_keygen.go is untouched); the EdDSA keygen path uses
// this shared helper rather than duplicate it again.
func (tKeyGen *TssKeyGen) keygenTimeoutBlame() error {
	tssConf := tKeyGen.tssCommonStruct.GetConf()
	blameMgr := tKeyGen.tssCommonStruct.GetBlameMgr()
	tKeyGen.logger.Error().Msgf("fail to generate message with %s", tssConf.KeyGenTimeout.String())
	lastMsg := blameMgr.GetLastMsg()
	failReason := blameMgr.GetBlame().FailReason
	if failReason == "" {
		failReason = blame.TssTimeout
	}
	if lastMsg == nil {
		tKeyGen.logger.Error().Msg("fail to start the keygen, the last produced message of this node is none")
		return errors.New("timeout before shared message is generated")
	}
	blameNodesUnicast, err := blameMgr.GetUnicastBlame(messages.KEYGEN2aUnicast)
	if err != nil {
		tKeyGen.logger.Error().Err(err).Msg("error in get unicast blame")
	}
	tKeyGen.tssCommonStruct.P2PPeersLock.RLock()
	threshold, err := conversion.GetThreshold(len(tKeyGen.tssCommonStruct.P2PPeers) + 1)
	tKeyGen.tssCommonStruct.P2PPeersLock.RUnlock()
	if err != nil {
		tKeyGen.logger.Error().Err(err).Msg("error in get the threshold to generate blame")
	}
	if len(blameNodesUnicast) > 0 && len(blameNodesUnicast) <= threshold {
		blameMgr.GetBlame().SetBlame(failReason, blameNodesUnicast, true, messages.KEYGEN2aUnicast)
	}
	blameNodesBroadcast, err := blameMgr.GetBroadcastBlame(lastMsg.Type())
	if err != nil {
		tKeyGen.logger.Error().Err(err).Msg("error in get broadcast blame")
	}
	blameMgr.GetBlame().AddBlameNodes(blameNodesBroadcast...)
	if len(blameMgr.GetBlame().BlameNodes) == 0 {
		blameNodesMisingShare, isUnicast, err := blameMgr.TssMissingShareBlame(messages.TSSKEYGENROUNDS)
		if err != nil {
			tKeyGen.logger.Error().Err(err).Msg("fail to get the node of missing share ")
		}
		if len(blameNodesMisingShare) > 0 && len(blameNodesMisingShare) <= threshold {
			blameMgr.GetBlame().AddBlameNodes(blameNodesMisingShare...)
			blameMgr.GetBlame().IsUnicast = isUnicast
		}
	}
	return nil
}

// GenerateNewKeyEdDSA runs an EdDSA (ed25519 / Edwards25519) threshold keygen ceremony. It mirrors the
// secp256k1 GenerateNewKey but uses tss-lib's eddsa/keygen party (which needs no Paillier pre-params)
// and saves the ed25519 keyshare into KeygenLocalState.EdDSALocalData. The caller MUST have set the
// tss-lib global curve to Edwards (see docs/architecture/stellar-eddsa-tss.md §9) — the ECDSA path is
// left byte-for-byte unchanged in tss_keygen.go.
func (tKeyGen *TssKeyGen) GenerateNewKeyEdDSA(keygenReq Request) (*bcrypto.ECPoint, error) {
	partiesID, localPartyID, err := conversion.GetParties(keygenReq.Keys, tKeyGen.localNodePubKey)
	if err != nil {
		return nil, fmt.Errorf("fail to get keygen parties: %w", err)
	}

	keyGenLocalStateItem := storage.KeygenLocalState{
		ParticipantKeys: keygenReq.Keys,
		LocalPartyKey:   tKeyGen.localNodePubKey,
		Algo:            storage.AlgoEdDSA,
	}

	threshold, err := conversion.GetThreshold(len(partiesID))
	if err != nil {
		return nil, err
	}
	keyGenPartyMap := new(sync.Map)
	ctx := btss.NewPeerContext(partiesID)
	params := btss.NewParameters(ctx, localPartyID, len(partiesID), threshold)
	outCh := make(chan btss.Message, len(partiesID))
	endCh := make(chan eddsakg.LocalPartySaveData, len(partiesID))
	errChan := make(chan struct{})
	blameMgr := tKeyGen.tssCommonStruct.GetBlameMgr()
	// EdDSA keygen takes no Paillier pre-params (unlike ECDSA).
	keyGenParty := eddsakg.NewLocalParty(params, outCh, endCh)
	partyIDMap := conversion.SetupPartyIDMap(partiesID)
	err1 := conversion.SetupIDMaps(partyIDMap, tKeyGen.tssCommonStruct.PartyIDtoP2PID)
	err2 := conversion.SetupIDMaps(partyIDMap, blameMgr.PartyIDtoP2PID)
	if err1 != nil || err2 != nil {
		tKeyGen.logger.Error().Msgf("error in creating mapping between partyID and P2P ID")
		return nil, err
	}
	// we never run multi keygen, so the moniker is set to default empty value
	keyGenPartyMap.Store("", keyGenParty)
	partyInfo := &common.PartyInfo{
		PartyMap:   keyGenPartyMap,
		PartyIDMap: partyIDMap,
	}

	tKeyGen.tssCommonStruct.SetPartyInfo(partyInfo)
	blameMgr.SetPartyInfo(keyGenPartyMap, partyIDMap)
	tKeyGen.tssCommonStruct.P2PPeersLock.Lock()
	tKeyGen.tssCommonStruct.P2PPeers = conversion.GetPeersID(tKeyGen.tssCommonStruct.PartyIDtoP2PID, tKeyGen.tssCommonStruct.GetLocalPeerID())
	tKeyGen.tssCommonStruct.P2PPeersLock.Unlock()
	var keyGenWg sync.WaitGroup
	keyGenWg.Add(2)
	// start keygen
	go func() {
		defer keyGenWg.Done()
		defer tKeyGen.logger.Debug().Msg(">>>>>>>>>>>>>.eddsa keyGenParty started")
		if err := keyGenParty.Start(); nil != err {
			tKeyGen.logger.Error().Err(err).Msg("fail to start eddsa keygen party")
			close(errChan)
		}
	}()
	go tKeyGen.tssCommonStruct.ProcessInboundMessages(tKeyGen.commStopChan, &keyGenWg)

	r, err := tKeyGen.processKeyGenEdDSA(errChan, outCh, endCh, keyGenLocalStateItem)
	if err != nil {
		close(tKeyGen.commStopChan)
		return nil, fmt.Errorf("fail to process eddsa keygen: %w", err)
	}
	select {
	case <-time.After(time.Second * 5):
		close(tKeyGen.commStopChan)
	case <-tKeyGen.tssCommonStruct.GetTaskDone():
		close(tKeyGen.commStopChan)
	}

	keyGenWg.Wait()
	return r, err
}

func (tKeyGen *TssKeyGen) processKeyGenEdDSA(errChan chan struct{},
	outCh <-chan btss.Message,
	endCh <-chan eddsakg.LocalPartySaveData,
	keyGenLocalStateItem storage.KeygenLocalState,
) (*bcrypto.ECPoint, error) {
	defer tKeyGen.logger.Debug().Msg("finished eddsa keygen process")
	tssConf := tKeyGen.tssCommonStruct.GetConf()
	blameMgr := tKeyGen.tssCommonStruct.GetBlameMgr()
	for {
		select {
		case <-errChan:
			tKeyGen.logger.Error().Msg("eddsa key gen failed")
			return nil, errors.New("error channel closed fail to start local party")

		case <-tKeyGen.stopChan:
			return nil, errors.New("received exit signal")

		case <-time.After(tssConf.KeyGenTimeout):
			if err := tKeyGen.keygenTimeoutBlame(); err != nil {
				return nil, err
			}
			return nil, blame.ErrTssTimeOut

		case msg := <-outCh:
			tKeyGen.logger.Debug().Msgf(">>>>>>>>>>eddsa msg: %s", msg.String())
			blameMgr.SetLastMsg(msg)
			if err := tKeyGen.tssCommonStruct.ProcessOutCh(msg, messages.TSSKeyGenMsg); err != nil {
				tKeyGen.logger.Error().Err(err).Msg("fail to process the message")
				return nil, err
			}

		case msg := <-endCh:
			tKeyGen.logger.Debug().Msgf("eddsa keygen finished successfully: %s", msg.EDDSAPub.Y().String())
			if err := tKeyGen.tssCommonStruct.NotifyTaskDone(); err != nil {
				tKeyGen.logger.Error().Err(err).Msg("fail to broadcast the keygen done")
			}
			pubKey, err := conversion.GetTssPubKeyEdDSA(msg.EDDSAPub)
			if err != nil {
				return nil, fmt.Errorf("fail to get eddsa pubkey: %w", err)
			}
			saved, err := json.Marshal(msg)
			if err != nil {
				return nil, fmt.Errorf("fail to marshal eddsa keyshare: %w", err)
			}
			keyGenLocalStateItem.EdDSALocalData = saved
			keyGenLocalStateItem.PubKey = pubKey
			if err := tKeyGen.stateManager.SaveLocalState(keyGenLocalStateItem); err != nil {
				return nil, fmt.Errorf("fail to save keygen result to storage: %w", err)
			}
			address := tKeyGen.p2pComm.ExportPeerAddress()
			if err := tKeyGen.stateManager.SaveAddressBook(address); err != nil {
				tKeyGen.logger.Error().Err(err).Msg("fail to save the peer addresses")
			}
			return msg.EDDSAPub, nil
		}
	}
}
