package keygen

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-log"

	"github.com/binance-chain/tss-lib/crypto"
	tcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/libp2p/go-libp2p-core/peer"

	btsskeygen "github.com/binance-chain/tss-lib/ecdsa/keygen"
	btss "github.com/binance-chain/tss-lib/tss"
	maddr "github.com/multiformats/go-multiaddr"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/messages"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/storage"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/common"
)

var (
	testPubKeys = []string{
		"tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
		"tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
		"tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
		"tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
	}
	testPriKeyArr = []string{
		"6LABmWB4iXqkqOJ9H0YFEA2CSSx6bA7XAKGyI/TDtas=",
		"528pkgjuCWfHx1JihEjiIXS7jfTS/viEdAbjqVvSifQ=",
		"JFB2LIJZtK+KasK00NcNil4PRJS4c4liOnK0nDalhqc=",
		"vLMGhVXMOXQVnAE3BUU8fwNj/q0ZbndKkwmxfS5EN9Y=",
	}

	testNodePrivkey = []string{
		"ZThiMDAxOTk2MDc4ODk3YWE0YThlMjdkMWY0NjA1MTAwZDgyNDkyYzdhNmMwZWQ3MDBhMWIyMjNmNGMzYjVhYg==",
		"ZTc2ZjI5OTIwOGVlMDk2N2M3Yzc1MjYyODQ0OGUyMjE3NGJiOGRmNGQyZmVmODg0NzQwNmUzYTk1YmQyODlmNA==",
		"MjQ1MDc2MmM4MjU5YjRhZjhhNmFjMmI0ZDBkNzBkOGE1ZTBmNDQ5NGI4NzM4OTYyM2E3MmI0OWMzNmE1ODZhNw==",
		"YmNiMzA2ODU1NWNjMzk3NDE1OWMwMTM3MDU0NTNjN2YwMzYzZmVhZDE5NmU3NzRhOTMwOWIxN2QyZTQ0MzdkNg==",
	}

	targets = []string{
		"16Uiu2HAmACG5DtqmQsHtXg4G2sLS65ttv84e7MrL4kapkjfmhxAp", "16Uiu2HAm4TmEzUqy3q3Dv7HvdoSboHk5sFj2FH3npiN5vDbJC6gh",
		"16Uiu2HAm2FzqoUdS6Y9Esg2EaGcAG5rVe1r6BFNnmmQr2H3bqafa",
	}
)

func TestPackage(t *testing.T) { TestingT(t) }

type TssKeygenTestSuite struct {
	comms        []*p2p.Communication
	preParams    []*btsskeygen.LocalPreParams
	partyNum     int
	stateMgrs    []storage.LocalStateManager
	nodePrivKeys []tcrypto.PrivKey
	targePeers   []peer.ID
}

var _ = Suite(&TssKeygenTestSuite{})

func (s *TssKeygenTestSuite) SetUpSuite(c *C) {
	common.InitLog("info", true, "keygen_test")
	conversion.SetupBech32Prefix()
	for _, el := range testNodePrivkey {
		priHexBytes, err := base64.StdEncoding.DecodeString(el)
		c.Assert(err, IsNil)
		rawBytes, err := hex.DecodeString(string(priHexBytes))
		c.Assert(err, IsNil)
		var priKey secp256k1.PrivKey
		priKey = rawBytes[:32]
		s.nodePrivKeys = append(s.nodePrivKeys, priKey)
	}

	for _, el := range targets {
		p, err := peer.Decode(el)
		c.Assert(err, IsNil)
		s.targePeers = append(s.targePeers, p)
	}
}

func (s *TssKeygenTestSuite) TearDownSuite(c *C) {
	for i := range s.comms {
		tempFilePath := path.Join(os.TempDir(), strconv.Itoa(i))
		err := os.RemoveAll(tempFilePath)
		c.Assert(err, IsNil)
	}
}

// SetUpTest set up environment for test key gen
func (s *TssKeygenTestSuite) SetUpTest(c *C) {
	ports := []int{
		18666, 18667, 18668, 18669,
	}
	s.partyNum = 4
	s.comms = make([]*p2p.Communication, s.partyNum)
	s.stateMgrs = make([]storage.LocalStateManager, s.partyNum)
	bootstrapPeer := "/ip4/127.0.0.1/tcp/18666/p2p/16Uiu2HAm4TmEzUqy3q3Dv7HvdoSboHk5sFj2FH3npiN5vDbJC6gh"
	multiAddr, err := maddr.NewMultiaddr(bootstrapPeer)
	c.Assert(err, IsNil)
	s.preParams = getPreparams(c)
	for i := 0; i < s.partyNum; i++ {
		buf, err := base64.StdEncoding.DecodeString(testPriKeyArr[i])
		c.Assert(err, IsNil)
		if i == 0 {
			comm, err := p2p.NewCommunication(&p2p.Config{Port: ports[i], RendezvousString: "asgard"}, nil)
			c.Assert(err, IsNil)
			c.Assert(comm.Start(buf[:]), IsNil)
			s.comms[i] = comm
			continue
		}
		comm, err := p2p.NewCommunication(&p2p.Config{Port: ports[i], RendezvousString: "asgard", BootstrapPeers: []maddr.Multiaddr{multiAddr}}, nil)
		c.Assert(err, IsNil)
		c.Assert(comm.Start(buf[:]), IsNil)
		s.comms[i] = comm
	}

	for i := 0; i < s.partyNum; i++ {
		baseHome := path.Join(os.TempDir(), strconv.Itoa(i))
		fMgr, err := storage.NewFileStateMgr(baseHome)
		c.Assert(err, IsNil)
		s.stateMgrs[i] = fMgr
	}
}

func (s *TssKeygenTestSuite) TearDownTest(c *C) {
	time.Sleep(time.Second)
	for _, item := range s.comms {
		c.Assert(item.Stop(), IsNil)
	}
}

func getPreparams(c *C) []*btsskeygen.LocalPreParams {
	const (
		testFileLocation = "../test_data"
		preParamTestFile = "preParam_test.data"
	)
	var preParamArray []*btsskeygen.LocalPreParams
	buf, err := ioutil.ReadFile(path.Join(testFileLocation, preParamTestFile))
	c.Assert(err, IsNil)
	preParamsStr := strings.Split(string(buf), "\n")
	for _, item := range preParamsStr {
		var preParam btsskeygen.LocalPreParams
		val, err := hex.DecodeString(item)
		c.Assert(err, IsNil)
		c.Assert(json.Unmarshal(val, &preParam), IsNil)
		preParamArray = append(preParamArray, &preParam)
	}
	return preParamArray
}

func (s *TssKeygenTestSuite) TestGenerateNewKey(c *C) {
	log.SetLogLevel("tss-lib", "info")
	sort.Strings(testPubKeys)
	req := NewRequest(testPubKeys, 10, "")
	messageID, err := common.MsgToHashString([]byte(strings.Join(req.Keys, "")))
	c.Assert(err, IsNil)
	conf := common.TssConfig{
		KeyGenTimeout:   120 * time.Second,
		KeySignTimeout:  120 * time.Second,
		PreParamTimeout: 5 * time.Second,
	}
	wg := sync.WaitGroup{}
	lock := &sync.Mutex{}
	keygenResult := make(map[int]*crypto.ECPoint)
	for i := 0; i < s.partyNum; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			comm := s.comms[idx]
			stopChan := make(chan struct{})
			localPubKey := testPubKeys[idx]
			keygenInstance := NewTssKeyGen(
				comm.GetLocalPeerID(),
				conf,
				localPubKey,
				comm.BroadcastMsgChan,
				stopChan,
				s.preParams[idx],
				messageID,
				s.stateMgrs[idx], s.nodePrivKeys[idx], s.comms[idx])
			c.Assert(keygenInstance, NotNil)
			keygenMsgChannel := keygenInstance.GetTssKeyGenChannels()
			comm.SetSubscribe(messages.TSSKeyGenMsg, messageID, keygenMsgChannel)
			comm.SetSubscribe(messages.TSSKeyGenVerMsg, messageID, keygenMsgChannel)
			comm.SetSubscribe(messages.TSSControlMsg, messageID, keygenMsgChannel)
			comm.SetSubscribe(messages.TSSTaskDone, messageID, keygenMsgChannel)
			defer comm.CancelSubscribe(messages.TSSKeyGenMsg, messageID)
			defer comm.CancelSubscribe(messages.TSSKeyGenVerMsg, messageID)
			defer comm.CancelSubscribe(messages.TSSControlMsg, messageID)
			defer comm.CancelSubscribe(messages.TSSTaskDone, messageID)
			resp, err := keygenInstance.GenerateNewKey(req)
			c.Assert(err, IsNil)
			lock.Lock()
			defer lock.Unlock()
			keygenResult[idx] = resp
		}(i)
	}
	wg.Wait()
	ans := keygenResult[0]
	for _, el := range keygenResult {
		c.Assert(el.Equals(ans), Equals, true)
	}
}

func (s *TssKeygenTestSuite) TestGenerateNewKeyWithStop(c *C) {
	conf := common.TssConfig{
		KeyGenTimeout:   20 * time.Second,
		KeySignTimeout:  20 * time.Second,
		PreParamTimeout: 5 * time.Second,
	}
	wg := sync.WaitGroup{}

	for i := 0; i < s.partyNum; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var localpubKey []string
			localpubKey = append(localpubKey, testPubKeys...)
			sort.Strings(testPubKeys)
			req := NewRequest(localpubKey, 10, "")
			messageID, err := common.MsgToHashString([]byte(strings.Join(req.Keys, "")))
			c.Assert(err, IsNil)
			comm := s.comms[idx]
			stopChan := make(chan struct{})
			localPubKey := testPubKeys[idx]
			keygenInstance := NewTssKeyGen(
				comm.GetLocalPeerID(),
				conf,
				localPubKey,
				comm.BroadcastMsgChan,
				stopChan,
				s.preParams[idx],
				messageID,
				s.stateMgrs[idx],
				s.nodePrivKeys[idx], s.comms[idx])
			c.Assert(keygenInstance, NotNil)
			keygenMsgChannel := keygenInstance.GetTssKeyGenChannels()
			comm.SetSubscribe(messages.TSSKeyGenMsg, messageID, keygenMsgChannel)
			comm.SetSubscribe(messages.TSSKeyGenVerMsg, messageID, keygenMsgChannel)
			comm.SetSubscribe(messages.TSSControlMsg, messageID, keygenMsgChannel)
			comm.SetSubscribe(messages.TSSTaskDone, messageID, keygenMsgChannel)
			defer comm.CancelSubscribe(messages.TSSKeyGenMsg, messageID)
			defer comm.CancelSubscribe(messages.TSSKeyGenVerMsg, messageID)
			defer comm.CancelSubscribe(messages.TSSControlMsg, messageID)
			defer comm.CancelSubscribe(messages.TSSTaskDone, messageID)
			if idx == 0 {
				go func() {
					time.Sleep(time.Millisecond * 200)
					close(keygenInstance.stopChan)
				}()
			}
			_, err = keygenInstance.GenerateNewKey(req)
			c.Assert(err, NotNil)
			// we skip the node 1 as we force it to stop
			if idx != 0 {
				blames := keygenInstance.GetTssCommonStruct().GetBlameMgr().GetBlame().BlameNodes
				c.Assert(blames, HasLen, 1)
				c.Assert(blames[0].Pubkey, Equals, testPubKeys[0])
			}
		}(i)
	}
	wg.Wait()
}

func (s *TssKeygenTestSuite) TestKeyGenWithError(c *C) {
	req := Request{
		Keys: testPubKeys[:],
	}
	conf := common.TssConfig{}
	stateManager := &storage.MockLocalStateManager{}
	keyGenInstance := NewTssKeyGen("", conf, "", nil, nil, nil, "test", stateManager, s.nodePrivKeys[0], nil)
	generatedKey, err := keyGenInstance.GenerateNewKey(req)
	c.Assert(err, NotNil)
	c.Assert(generatedKey, IsNil)
}

func (s *TssKeygenTestSuite) TestCloseKeyGenNotifyChannel(c *C) {
	conf := common.TssConfig{}
	stateManager := &storage.MockLocalStateManager{}
	keyGenInstance := NewTssKeyGen("", conf, "", nil, nil, nil, "test", stateManager, s.nodePrivKeys[0], s.comms[0])

	taskDone := messages.TssTaskNotifier{TaskDone: true}
	taskDoneBytes, err := json.Marshal(taskDone)
	c.Assert(err, IsNil)

	msg := &messages.WrappedMessage{
		MessageType: messages.TSSTaskDone,
		MsgID:       "test",
		Payload:     taskDoneBytes,
	}
	partyIdMap := make(map[string]*btss.PartyID)
	partyIdMap["1"] = nil
	partyIdMap["2"] = nil
	fakePartyInfo := &common.PartyInfo{
		PartyMap:   nil,
		PartyIDMap: partyIdMap,
	}
	keyGenInstance.tssCommonStruct.SetPartyInfo(fakePartyInfo)
	err = keyGenInstance.tssCommonStruct.ProcessOneMessage(msg, "node1")
	c.Assert(err, IsNil)
	err = keyGenInstance.tssCommonStruct.ProcessOneMessage(msg, "node2")
	c.Assert(err, IsNil)
	err = keyGenInstance.tssCommonStruct.ProcessOneMessage(msg, "node1")
	c.Assert(err, ErrorMatches, "duplicated notification from peer node1 ignored")
}
