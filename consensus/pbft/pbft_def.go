// The pbft consensus process

package pbft

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/Aj002Th/BlockchainEmulator/consensus/pbft/meter"
	"github.com/Aj002Th/BlockchainEmulator/data/base"
	"github.com/Aj002Th/BlockchainEmulator/data/chain"
	"github.com/Aj002Th/BlockchainEmulator/misc"
	"github.com/Aj002Th/BlockchainEmulator/network"
	"github.com/Aj002Th/BlockchainEmulator/params"
	"github.com/Aj002Th/BlockchainEmulator/storage/blockStorage"
	"github.com/Aj002Th/BlockchainEmulator/storage/stateStorage"
)

type ConsensusNode struct {
	// the local config about pbft
	RunningNode *Node  // the node information
	ShardID     uint64 // denote the ID of the shard (or pbft), only one pbft consensus in a shard
	NodeID      uint64 // denote the ID of the node in the pbft (shard)

	// the data structure for blockchain
	CurChain *chain.BlockChain // all node in the shard maintain the same blockchain
	db       blockStorage.BlockStorage
	sb       stateStorage.StateStorage

	// the global config about pbft
	pbftChainConfig *chain.Config                // the chain config in this pbft
	ipNodeTable     map[uint64]map[uint64]string // denote the ip of the specific node
	nodeNums        uint64                       // the number of nodes in this pbft, denoted by N
	maliciousNums   uint64                       // f, 3f + 1 = N
	view            uint64                       // denote the view of this pbft, the main node can be inferred from this variant

	// the control message and message checking utils in pbft
	sequenceID        uint64                    // the message sequence id of the pbft
	stop              bool                      // send stop signal
	pStop             chan uint64               // channel for stopping consensus
	requestPool       map[string]*Request       // RequestHash to Request
	cntPrepareConfirm map[string]map[*Node]bool // count the prepare confirm message, [messageHash][Node]bool
	cntCommitConfirm  map[string]map[*Node]bool // count the commit confirm message, [messageHash][Node]bool
	isCommitBroadcast map[string]bool           // denote whether the commit is broadcast
	isReply           map[string]bool           // denote whether the message is reply message
	height2Digest     map[uint64]string         // sequence (block height) -> request, fast read

	// locks about pbft
	sequenceLock sync.Mutex // the lock of sequence
	lock         sync.Mutex // lock the stage
	askForLock   sync.Mutex // lock for asking for a series of requests
	stopLock     sync.Mutex // lock the stop variant

	// seqID of other Shards, to synchronize
	seqIDMap   map[uint64]uint64
	seqMapLock sync.Mutex

	// logger
	pl *misc.PbftLog

	// to handle the message in the pbft
	ihm ExtraOpInConsensus

	// to handle the message outside pbft
	ohm OpInterShards
}

// NewPbftNode generate a pbft consensus for a node
func NewPbftNode(nodeID uint64, pcc *chain.Config) *ConsensusNode {
	self := new(ConsensusNode)
	self.ipNodeTable = params.IPmapNodeTable
	self.nodeNums = pcc.NodesNum
	self.ShardID = 0
	self.NodeID = nodeID
	self.pbftChainConfig = pcc

	var err error
	self.db = blockStorage.NewBoltStorage(uint(nodeID))
	self.sb = stateStorage.NewMemKVStore()
	self.CurChain, err = chain.NewBlockChain(pcc, self.sb, self.db)
	if err != nil {
		log.Panic("cannot new a blockchain")
	}

	self.RunningNode = &Node{
		NodeID:  nodeID,
		ShardID: 0,
		IPaddr:  self.ipNodeTable[0][nodeID],
	}

	self.stop = false
	self.sequenceID = self.CurChain.CurrentBlock.Header.Number + 1
	self.pStop = make(chan uint64)
	self.requestPool = make(map[string]*Request)
	self.cntPrepareConfirm = make(map[string]map[*Node]bool)
	self.cntCommitConfirm = make(map[string]map[*Node]bool)
	self.isCommitBroadcast = make(map[string]bool)
	self.isReply = make(map[string]bool)
	self.height2Digest = make(map[uint64]string)
	self.maliciousNums = (self.nodeNums - 1) / 3
	self.view = 0
	self.seqIDMap = make(map[uint64]uint64)
	self.pl = misc.NewPbftLog(0, nodeID)

	base.NodeLog = self.pl

	// choose how to handle the messages in pbft or beyond pbft
	self.ihm = &RawPbftPbftExtraHandleMod{
		node: self,
	}
	self.ohm = &RawPbftOutsideModule{
		node: self,
	}

	return self
}

func dispatchHelper[T any](content []byte, handler func(*T)) {
	message := new(T)
	err := json.Unmarshal(content, message)
	if err != nil {
		log.Panic(err)
	}
	handler(message)
}

// handle the raw message, send it to corresponded interfaces
// 还没反序列化。根据他的奇葩协议要做派发。不是在这里反序列化的。
func (self *ConsensusNode) dispatchMessage(msg []byte) {
	msgType, content := SplitMessage(msg)
	if len(content) > 2000 {
		self.pl.Printf("Received a %v: %v\n", msgType, string(content[:2000]))
	} else {
		self.pl.Printf("Received a %v: %v\n", msgType, string(content))
	}
	switch msgType {
	// pbft inside message type
	case CPrePrepare:
		dispatchHelper(content, self.handlePrePrepare) // 竟然能做type inference， 牛逼。
	case CPrepare:
		dispatchHelper(content, self.handlePrepare)
	case CCommit:
		dispatchHelper(content, self.handleCommit)
	case CStop:
		self.setStopAndCleanUp()

	// handle the message from outside
	default:
		self.ohm.HandleMessageOutsidePBFT(msgType, content)
	}
}

func (self *ConsensusNode) Run() {
	meter.NodeSideStart()
	if self.NodeID == 0 {
		// 主节点需要负责从 txPool 中取出 tx 来处理
		println("well my node_id is 0 so i will do propose")
		go self.doPropose()

		// 主节点需要向 supervisor 持续报告 keepAlive
		go func() {
			for {
				b := KeepAliveMsg{Msg: "alive"}
				m, err := json.Marshal(b)
				if err != nil {
					panic(err)
				}
				MergeAndSend(CKeepAlive, m, params.SupervisorEndpoint, self.pl)
				time.Sleep(time.Second * 5)
			}
		}()
	}
	self.serve()
}

// 起一个TCP Server端。
func (self *ConsensusNode) serve() {
	ch := network.Tcp.Serve(self.RunningNode.IPaddr)
	for {
		clientRequest, ok := <-ch
		if !ok {
			self.pl.Println("Pbft Node, the Tcp channel is closed")
			return
		}
		self.dispatchMessage(clientRequest)
	}
}

// when received stop
func (self *ConsensusNode) setStopAndCleanUp() {
	self.pl.Println("handling stop message")
	self.stopLock.Lock()
	self.stop = true
	self.stopLock.Unlock()
	if self.NodeID == self.view {
		self.pStop <- 1
	}

	self.pl.Println("Before GatherAndSend")

	GatherAndSend(int(self.NodeID), self.pl)

	network.Tcp.Close()
	self.CurChain.CloseBlockChain()
	self.pl.Println("handled stop message")
}

func GatherAndSend(nodeID int, pl *log.Logger) {
	// Process相关
	b := BookingMsg{
		AvgCpuTime:    meter.AvgCpuPercent,
		DiskMetric:    meter.DiskMetric,
		TotalUpload:   meter.TotalUpload,
		TotalDownload: meter.TotalDownload,
		TotalTime:     int64(time.Since(meter.TimeBegin)),
		NodeId:        nodeID,
	}
	m, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	MergeAndSend(CBooking, m, params.SupervisorEndpoint, pl)
}

// this func is only invoked by main node
func (self *ConsensusNode) doPropose() {
	if self.view != self.NodeID { // 保证只能主节点调用。否则返回。
		return
	}
	for { // for循环一直在do Propose。
		select { // 判断是否停止的那个信号量。
		case <-self.pStop:
			self.pl.Printf("S%dN%d stop...\n", self.ShardID, self.NodeID)
			return
		default:
		}
		time.Sleep(time.Duration(int64(params.BlockInterval)) * time.Millisecond) // 不停睡觉-propose循环。（发PrePrepare）

		self.sequenceLock.Lock()
		self.pl.Printf("S%dN%d get sequenceLock locked, now trying to propose...\n", self.ShardID, self.NodeID)

		// propose
		// implement interface to generate propose
		_, r := self.ihm.HandleInPropose()

		digest := getDigest(r)
		self.requestPool[string(digest)] = r
		self.pl.Printf("S%dN%d put the request into the pool ...\n", self.ShardID, self.NodeID)

		ppmsg := PrePrepare{
			RequestMsg: r,
			Digest:     digest,
			SeqID:      self.sequenceID,
		}
		self.height2Digest[self.sequenceID] = string(digest)

		// marshal and broadcast
		ppbyte, err := json.Marshal(ppmsg)
		if err != nil {
			log.Panic()
		}
		MergeAndBroadcast(CPrePrepare, ppbyte, self.RunningNode.IPaddr, self.getNeighborNodes(), self.pl)
	}
}

func MergeAndBroadcast(t MessageType, data []byte, from string, to []string, logger *log.Logger) {
	if len(data) < 2000 {
		logger.Printf("Broadcasting a %v: %v", t, string(data))
	} else {
		logger.Printf("Broadcasting a %v: %v", t, string(data[:2000]))
	}
	msgSend := MergeMessage(t, data)
	network.Tcp.Broadcast(from, to, msgSend)
}
