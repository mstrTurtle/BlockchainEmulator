// addtional module for new consensus
package pbft

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Aj002Th/BlockchainEmulator/data/base"
	"github.com/Aj002Th/BlockchainEmulator/data/chain"
	"github.com/Aj002Th/BlockchainEmulator/logger"
	"github.com/Aj002Th/BlockchainEmulator/params"
)

// RawPbftPbftExtraHandleMod simple implementation of pbftHandleModule interface ...
// only for block request and use transaction pbft
type RawPbftPbftExtraHandleMod struct {
	// pointer to pbft data
	node *ConsensusNode
}

// HandleInPropose propose request with different types
// 调用GenerateBlock产生一个区块。从交易池子里取出至多n个交易，并且生成一个新区块。
func (self *RawPbftPbftExtraHandleMod) HandleInPropose() (bool, *Request) {
	// new blocks
	block := self.node.CurChain.GenerateBlock()
	r := &Request{
		RequestType: BlockRequest,
		ReqTime:     time.Now(),
	}
	r.Msg.Content = block.Encode()

	return true, r
}

// the diy operation in preprepare
func (self *RawPbftPbftExtraHandleMod) HandleInPrePrepare(ppmsg *PrePrepare) bool {
	if self.node.CurChain.IsValidBlock(base.DecodeBlock(ppmsg.RequestMsg.Msg.Content)) != nil {
		self.node.pl.Printf("S%dN%d : not a valid block\n", self.node.ShardID, self.node.NodeID)
		return false
	}
	self.node.pl.Printf("S%dN%d : the pre-prepare pbft is correct, putting it into the RequestPool. \n", self.node.ShardID, self.node.NodeID)
	self.node.requestPool[string(ppmsg.Digest)] = ppmsg.RequestMsg
	// merge to be a prepare pbft
	return true
}

// the operation in prepare, and in pbft + tx pbfting, this function does not need to do any.
func (self *RawPbftPbftExtraHandleMod) HandleInPrepare(pmsg *Prepare) bool {
	logger.Println("No operations are performed in Extra handle mod")
	return true
}

// print the details of a blockchain
func PrintBlockChain(bc *chain.BlockChain) string {
	vals := []interface{}{
		bc.CurrentBlock.Header.Number,
		bc.CurrentBlock.Hash,
		bc.CurrentBlock.Header.StateRoot,
		bc.CurrentBlock.Header.Time,
		bc.BlockStorage,
	}
	res := fmt.Sprintf("%v\n", vals)
	logger.Println(res)
	return res
}

// the operation in commit.
func (self *RawPbftPbftExtraHandleMod) HandleInCommit(cmsg *Commit) bool {
	r := self.node.requestPool[string(cmsg.Digest)]
	// requestType ...
	block := base.DecodeBlock(r.Msg.Content)
	self.node.pl.Printf("S%dN%d : adding the block %d...now height = %d \n", self.node.ShardID, self.node.NodeID, block.Header.Number, self.node.CurChain.CurrentBlock.Header.Number)
	self.node.CurChain.AddBlock(block)
	self.node.pl.Printf("S%dN%d : added the block %d... \n", self.node.ShardID, self.node.NodeID, block.Header.Number)
	PrintBlockChain(self.node.CurChain)

	// now try to pbft txs to other shards (for main nodes)
	if self.node.NodeID == self.node.view {
		self.node.pl.Printf("S%dN%d : main node is trying to send pbft txs at height = %d \n", self.node.ShardID, self.node.NodeID, block.Header.Number)
		// generate pbft pool and collect txs excuted
		txExcuted := make([]*base.Transaction, 0)
		for _, tx := range block.Body {
			txExcuted = append(txExcuted, tx)
		}
		// send txs excuted in this block to the listener
		// add more pbft to measure more metrics
		bim := BlockInfoMsg{
			BlockBodyLength: len(block.Body),
			ExcutedTxs:      txExcuted,
			Epoch:           0,
			SenderShardID:   self.node.ShardID,
			ProposeTime:     r.ReqTime,
			CommitTime:      time.Now(),
			TxpoolSize:      len(self.node.CurChain.TransactionPool.Queue),
		}
		bByte, err := json.Marshal(bim)
		if err != nil {
			log.Panic()
		}
		MergeAndSend(CBlockInfo, bByte, params.SupervisorEndpoint, self.node.pl)
		self.node.pl.Printf("S%dN%d : sended excuted txs\n", self.node.ShardID, self.node.NodeID)
	}
	return true
}
