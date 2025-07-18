package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/addr"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/messages"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/storage"
)

var (
	joinPartyProtocol           protocol.ID = "/p2p/join-party"
	joinPartyProtocolWithLeader protocol.ID = "/p2p/join-party-leader"
)

// TSSProtocolID protocol id used for tss
var (
	TSSProtocolID        protocol.ID = "/p2p/tss"
	ObservedTxProtocolID protocol.ID = "/p2p/observed-tx"
)

const (
	// TimeoutConnecting maximum time for wait for peers to connect
	TimeoutConnecting = time.Second * 20

	StreamUnknown = "UNKNOWN"
	StreamMsgDone = "done"
)

// Message that get transfer across the wire
type Message struct {
	PeerID  peer.ID
	Payload []byte
}

// Communication use p2p to broadcast messages among all the TSS nodes
type Communication struct {
	config           P2PConfig
	stateManager     *storage.FileStateMgr
	logger           zerolog.Logger
	listenAddr       maddr.Multiaddr
	host             host.Host
	wg               *sync.WaitGroup
	stopChan         chan struct{} // channel to indicate whether we should stop
	subscribers      map[messages.THORChainTSSMessageType]*MessageIDSubscriber
	subscriberLocker *sync.Mutex
	streamCount      int64
	BroadcastMsgChan chan *messages.BroadcastMsgChan
	externalAddr     maddr.Multiaddr
	streamMgr        *StreamMgr
}

type P2PConfig interface {
	GetBootstrapPeers() ([]maddr.Multiaddr, error)
	GetP2PPort() int
	GetExternalIP() string
	GetRendezvous() string
}

func StartP2P(
	cfg P2PConfig,
	priKey tcrypto.PrivKey,
	baseFolder string,
) (*Communication, *storage.FileStateMgr, error) {
	stateManager, err := storage.NewFileStateMgr(baseFolder)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to create file state manager")
	}

	comm, err := NewCommunication(cfg, stateManager)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to create communication layer: %w", err)
	}

	priKeyRawBytes, err := conversion.GetPriKeyRawBytes(priKey)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to get private key")
	}
	if err := comm.Start(priKeyRawBytes); nil != err {
		return nil, nil, fmt.Errorf("fail to start p2p network: %w", err)
	}

	return comm, stateManager, nil
}

// NewCommunication create a new instance of Communication
func NewCommunication(cfg P2PConfig, stateManager *storage.FileStateMgr) (*Communication, error) {
	port, externalIP := cfg.GetP2PPort(), cfg.GetExternalIP()
	addr, err := maddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, fmt.Errorf("fail to create listen addr: %w", err)
	}
	var externalAddr maddr.Multiaddr = nil
	if len(externalIP) != 0 {
		externalAddr, err = maddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", externalIP, port))
		if err != nil {
			return nil, fmt.Errorf("fail to create listen with given external IP: %w", err)
		}
	}
	return &Communication{
		config:           cfg,
		stateManager:     stateManager,
		logger:           log.With().Str("module", "communication").Logger(),
		listenAddr:       addr,
		wg:               &sync.WaitGroup{},
		stopChan:         make(chan struct{}),
		subscribers:      make(map[messages.THORChainTSSMessageType]*MessageIDSubscriber),
		subscriberLocker: &sync.Mutex{},
		streamCount:      0,
		BroadcastMsgChan: make(chan *messages.BroadcastMsgChan, 1024),
		externalAddr:     externalAddr,
		streamMgr:        NewStreamMgr(),
	}, nil
}

// GetHost return the host
func (c *Communication) GetHost() host.Host {
	return c.host
}

// GetLocalPeerID from p2p host
func (c *Communication) GetLocalPeerID() string {
	return c.host.ID().String()
}

// Broadcast message to Peers
func (c *Communication) Broadcast(peers []peer.ID, msg []byte, msgID string) {
	if len(peers) == 0 {
		return
	}
	// try to discover all peers and then broadcast the messages
	c.wg.Add(1)
	go c.broadcastToPeers(peers, msg, msgID)
}

func (c *Communication) broadcastToPeers(peers []peer.ID, msg []byte, msgID string) {
	defer c.wg.Done()
	defer func() {
		c.logger.Debug().Msgf("finished sending message to peer(%v)", peers)
	}()
	var wgSend sync.WaitGroup
	wgSend.Add(len(peers))
	for _, p := range peers {
		go func(p peer.ID) {
			defer wgSend.Done()
			if err := c.writeToStream(p, msg, msgID); nil != err {
				c.logger.Error().Err(err).Msg("fail to write to stream")
			}
		}(p)
	}
	wgSend.Wait()
}

func (c *Communication) writeToStream(pID peer.ID, msg []byte, msgID string) error {
	// don't send to ourselves
	if pID == c.host.ID() {
		return nil
	}
	stream, err := c.connectToOnePeer(pID)
	if err != nil {
		return fmt.Errorf("fail to open stream to peer(%s): %w", pID, err)
	}
	if nil == stream {
		return nil
	}

	defer func() {
		c.streamMgr.AddStream(msgID, stream)
	}()
	c.logger.Debug().Msgf(">>>writing messages to peer(%s)", pID)

	return WriteStreamWithBuffer(msg, stream)
}

func (c *Communication) handleStreamTss(stream network.Stream) {
	peerID := stream.Conn().RemotePeer().String()
	c.logger.Debug().Msgf("reading from tss stream of peer: %s", peerID)

	select {
	case <-c.stopChan:
		return
	default:
		dataBuf, err := ReadStreamWithBuffer(stream)
		if err != nil {
			c.logger.Error().Err(err).Msgf("fail to read from stream,peerID: %s", peerID)
			c.streamMgr.AddStream(StreamUnknown, stream)
			return
		}
		var wrappedMsg messages.WrappedMessage
		if err := json.Unmarshal(dataBuf, &wrappedMsg); nil != err {
			c.logger.Error().Err(err).Msg("fail to unmarshal wrapped message bytes")
			c.streamMgr.AddStream(StreamUnknown, stream)
			return
		}
		c.logger.Debug().Msgf(">>>>>>>[%s] %s", wrappedMsg.MessageType, string(wrappedMsg.Payload))
		c.streamMgr.AddStream(wrappedMsg.MsgID, stream)
		channel := c.getSubscriber(wrappedMsg.MessageType, wrappedMsg.MsgID)
		if nil == channel {
			c.logger.Debug().Msgf("no MsgID %s found for this message", wrappedMsg.MsgID)
			c.logger.Debug().Msgf("no MsgID %s found for this message", wrappedMsg.MessageType)
			return
		}
		channel <- &Message{
			PeerID:  stream.Conn().RemotePeer(),
			Payload: dataBuf,
		}

	}
}

func (c *Communication) getPeers() addr.AddrList {
	var bootstrapPeers addr.AddrList

	cfgBoostrapPeers, err := c.config.GetBootstrapPeers()
	if err != nil {
		c.logger.Error().Err(err).Msg("fail to get bootstrap peers from config")
	} else {
		bootstrapPeers = cfgBoostrapPeers
	}

	if c.stateManager != nil {
		savedPeers, err := c.stateManager.RetrieveP2PAddresses()
		if err != nil {
			c.logger.Error().Err(err).Msg("fail to get saved peers from state manager")
		} else {
			bootstrapPeers = append(bootstrapPeers, savedPeers...)
		}
	}

	return bootstrapPeers
}

func (c *Communication) bootStrapConnectivityCheck() error {
	bootstrapPeers := c.getPeers()

	if len(bootstrapPeers) == 0 {
		c.logger.Error().Msg("we do not have the bootstrap node set, quit the connectivity check")
		return nil
	}

	var onlineNodes uint32
	var wg sync.WaitGroup
	for _, el := range bootstrapPeers {
		peer, err := peer.AddrInfoFromP2pAddr(el)
		if err != nil {
			c.logger.Error().Err(err).Msg("error in decode the bootstrap node, skip it")
			continue
		}
		wg.Add(1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			defer wg.Done()
			outChan := ping.Ping(ctx, c.host, peer.ID)
			select {
			case ret, ok := <-outChan:
				if !ok {
					return
				}
				if ret.Error == nil {
					c.logger.Debug().Msgf("connect to peer %v with RTT %v\n", peer.ID, ret.RTT)
					atomic.AddUint32(&onlineNodes, 1)
				}
			case <-ctx.Done():
				c.logger.Error().Msgf("fail to ping the node %s within 2 seconds", peer.ID)
			}
		}()
	}
	wg.Wait()

	if onlineNodes > 0 {
		c.logger.Info().Msgf("we have successfully ping pong %d nodes", onlineNodes)
		return nil
	}
	c.logger.Error().Msg("fail to ping any bootstrap node")
	return errors.New("the node cannot ping any bootstrap node")
}

func (c *Communication) startChannel(privKeyBytes []byte) error {
	ctx := context.Background()
	p2pPriKey, err := crypto.UnmarshalSecp256k1PrivateKey(privKeyBytes)
	if err != nil {
		c.logger.Error().Msgf("error is %f", err)
		return err
	}

	addressFactory := func(addrs []maddr.Multiaddr) []maddr.Multiaddr {
		if c.externalAddr != nil {
			return []maddr.Multiaddr{c.externalAddr}
		}
		return addrs
	}

	h, err := libp2p.New(ctx,
		libp2p.ListenAddrs([]maddr.Multiaddr{c.listenAddr}...),
		libp2p.Identity(p2pPriKey),
		libp2p.AddrsFactory(addressFactory),
	)
	if err != nil {
		return fmt.Errorf("fail to create p2p host: %w", err)
	}
	c.host = h
	c.logger.Info().Msgf("Host created, we are: %s, at: %s", h.ID(), h.Addrs())
	h.SetStreamHandler(TSSProtocolID, c.handleStreamTss)
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return fmt.Errorf("fail to create DHT: %w", err)
	}
	c.logger.Debug().Msg("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("fail to bootstrap DHT: %w", err)
	}

	var connectionErr error
	for i := 0; i < 5; i++ {
		connectionErr = c.connectToBootstrapPeers()
		if connectionErr == nil {
			break
		}
		c.logger.Error().Msg("cannot connect to any bootstrap node, retry in 5 seconds")
		time.Sleep(time.Second * 5)
	}
	if connectionErr != nil {
		return fmt.Errorf("fail to connect to bootstrap peer: %w", connectionErr)
	}

	// We use a rendezvous point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, c.config.GetRendezvous())
	err = c.bootStrapConnectivityCheck()
	if err != nil {
		return err
	}

	c.logger.Info().Msg("Successfully announced!")
	return nil
}

func (c *Communication) connectToOnePeer(pID peer.ID) (network.Stream, error) {
	c.logger.Debug().Msgf("peer:%s,current:%s", pID, c.host.ID())
	// dont connect to itself
	if pID == c.host.ID() {
		return nil, nil
	}
	c.logger.Debug().Msgf("connect to peer : %s", pID.String())
	ctx, cancel := context.WithTimeout(context.Background(), TimeoutConnecting)
	defer cancel()
	stream, err := c.host.NewStream(ctx, pID, TSSProtocolID)
	if err != nil {
		return nil, fmt.Errorf("fail to create new stream to peer: %s, %w", pID, err)
	}
	return stream, nil
}

func (c *Communication) connectToBootstrapPeers() error {
	bootstrapPeers := c.getPeers()
	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	if len(bootstrapPeers) == 0 {
		c.logger.Info().Msg("no bootstrap node set, we skip the connection")
		return nil
	}
	var wg sync.WaitGroup
	connRet := make(chan bool, len(bootstrapPeers))
	for _, peerAddr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return fmt.Errorf("fail to add peer: %w", err)
		}
		wg.Add(1)
		go func(connRet chan bool) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), TimeoutConnecting)
			defer cancel()
			if err := c.host.Connect(ctx, *pi); err != nil {
				c.logger.Error().Err(err).Msgf("fail to connect to %s", pi.String())
				connRet <- false
				return
			}
			connRet <- true
			c.logger.Info().Msgf("Connection established with bootstrap node: %s", *pi)
		}(connRet)
	}
	wg.Wait()
	for i := 0; i < len(bootstrapPeers); i++ {
		if <-connRet {
			return nil
		}
	}
	return errors.New("fail to connect to any peer")
}

// Start will start the communication
func (c *Communication) Start(priKeyBytes []byte) error {
	err := c.startChannel(priKeyBytes)
	if err == nil {
		c.wg.Add(1)
		go c.ProcessBroadcast()
	}
	return err
}

// Stop communication
func (c *Communication) Stop() error {
	// we need to stop the handler and the p2p services firstly, then terminate the our communication threads
	if err := c.host.Close(); err != nil {
		c.logger.Err(err).Msg("fail to close host network")
	}

	close(c.stopChan)
	c.wg.Wait()
	return nil
}

func (c *Communication) SetSubscribe(topic messages.THORChainTSSMessageType, msgID string, channel chan *Message) {
	c.subscriberLocker.Lock()
	defer c.subscriberLocker.Unlock()

	messageIDSubscribers, ok := c.subscribers[topic]
	if !ok {
		messageIDSubscribers = NewMessageIDSubscriber()
		c.subscribers[topic] = messageIDSubscribers
	}
	messageIDSubscribers.Subscribe(msgID, channel)
}

func (c *Communication) getSubscriber(topic messages.THORChainTSSMessageType, msgID string) chan *Message {
	c.subscriberLocker.Lock()
	defer c.subscriberLocker.Unlock()
	messageIDSubscribers, ok := c.subscribers[topic]
	if !ok {
		c.logger.Debug().Msgf("fail to find subscribers for %s", topic)
		return nil
	}
	return messageIDSubscribers.GetSubscriber(msgID)
}

func (c *Communication) CancelSubscribe(topic messages.THORChainTSSMessageType, msgID string) {
	c.subscriberLocker.Lock()
	defer c.subscriberLocker.Unlock()

	messageIDSubscribers, ok := c.subscribers[topic]
	if !ok {
		c.logger.Debug().Msgf("cannot find the given channels %s", topic.String())
		return
	}
	if nil == messageIDSubscribers {
		return
	}
	messageIDSubscribers.UnSubscribe(msgID)
	if messageIDSubscribers.IsEmpty() {
		delete(c.subscribers, topic)
	}
}

func (c *Communication) ProcessBroadcast() {
	c.logger.Debug().Msg("start to process broadcast message channel")
	defer c.logger.Debug().Msg("stop process broadcast message channel")
	defer c.wg.Done()
	for {
		select {
		case msg := <-c.BroadcastMsgChan:
			wrappedMsgBytes, err := json.Marshal(msg.WrappedMessage)
			if err != nil {
				c.logger.Error().Err(err).Msg("fail to marshal a wrapped message to json bytes")
				continue
			}
			c.logger.Debug().Msgf("broadcast message %s to %+v", msg.WrappedMessage, msg.PeersID)
			c.Broadcast(msg.PeersID, wrappedMsgBytes, msg.WrappedMessage.MsgID)

		case <-c.stopChan:
			return
		}
	}
}

func (c *Communication) ReleaseStream(msgID string) {
	c.streamMgr.ReleaseStream(msgID)
}
