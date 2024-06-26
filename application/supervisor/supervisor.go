// Supervisor is an abstract role in this simulator that may read txs, generate partition infos,
// and handle history data.

package supervisor

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/committee"
	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/meter"
	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/metrics"
	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/signal"
	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/supervisor_log"
	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/webapi"
	"github.com/Aj002Th/BlockchainEmulator/consensus/pbft"
	"github.com/Aj002Th/BlockchainEmulator/network"
	"github.com/Aj002Th/BlockchainEmulator/params"
	sig "github.com/Aj002Th/BlockchainEmulator/signal"
)

type Supervisor struct {
	IpNodeTable map[uint64]map[uint64]string // basic infos
	// tcpLn             net.Listener                  // tcp control
	tcpLock           sync.Mutex                    // listenStop bool
	sl                *supervisor_log.SupervisorLog // logger module
	Ss                *signal.StopSignal            // control components// to control the stop message sending
	cmt               committee.CommitteeModule     // supervisor and committee components
	blockPostedSignal sig.Signal[pbft.BlockInfoMsg] // 内部信号

	txCompleteCount int
	pbftItems       []webapi.PbftItem
	OnNodeStart     sig.Signal[struct{}]

	cntBooking int
	bookings   []pbft.BookingMsg
	result     chan []metrics.Desc

	waitOnce        func()
	waitMasterReady chan struct{}
}

func NewSupervisor() *Supervisor {
	d := &Supervisor{}
	d.IpNodeTable = params.IPmapNodeTable
	d.sl = supervisor_log.NewSupervisorLog()
	d.Ss = signal.NewStopSignal(2 * int(1))
	d.blockPostedSignal = sig.NewAsyncSignalImpl[pbft.BlockInfoMsg]("xx")
	d.cmt = committee.NewPbftCommitteeModule(d.IpNodeTable, d.Ss, d.sl, params.FileInput, params.TotalDataSize, params.BatchSize)
	d.txCompleteCount = 0

	d.OnNodeStart = sig.GetSignalByName[struct{}]("OnNodeStart")

	d.result = make(chan []metrics.Desc)

	// 主节点 keep alive 相关处理
	d.waitMasterReady = make(chan struct{})
	d.waitOnce = sync.OnceFunc(func() {
		println("OnceFunc called")
		d.waitMasterReady <- struct{}{}
	})

	return d
}

// 根据Body长度决定是否Inc.
// 并且触发测量模块。
func (d *Supervisor) handleBlockInfoMsg(m *pbft.BlockInfoMsg) {
	supervisor_log.DebugLog.Println("in handleBlockInfos")

	supervisor_log.DebugLog.Println("StopSignal Check")

	// StopSignal check
	if m.BlockBodyLength == 0 {
		supervisor_log.DebugLog.Println("BodyLength == 0, Inc")
		d.Ss.StopGapInc()
	} else {
		supervisor_log.DebugLog.Println("BodyLength != 0, Reset")
		d.Ss.StopGapReset()
	}

	supervisor_log.DebugLog.Printf("received from shard %d in epoch %d.\n", m.SenderShardID, m.Epoch)

	d.txCompleteCount += len(m.ExcutedTxs)
	webapi.GlobalProxy.Enqueue(webapi.Computing(params.TotalDataSize, d.txCompleteCount))

	pbftItem := webapi.PbftItem{TxpoolSize: int(m.TxpoolSize), Tx: len(m.ExcutedTxs)}
	d.pbftItems = append(d.pbftItems, pbftItem)
	// measure update
	d.blockPostedSignal.Emit(*m)
	// add codes here ...
}

func (d *Supervisor) handleBookingMsg(m *pbft.BookingMsg) {
	d.cntBooking++
	d.bookings = append(d.bookings, *m)
	d.sl.Slog.Printf("handleBookingMsg, cnt = %v\n", d.cntBooking)
	if d.cntBooking == params.NodeNum {
		result := meter.GetResult(&d.bookings)
		d.sl.Slog.Printf("handleBookingMsg now got result: %v\n", result)
		d.sl.Slog.Printf("handleBookingMsg now writing to channel\n")
		d.result <- result
	}
}

func (d *Supervisor) handleKeepAliveMsg(m *pbft.KeepAliveMsg) {
	d.waitOnce()
}

func (d *Supervisor) Wait() {
	<-d.waitMasterReady
	time.Sleep(time.Second)
}

func (d *Supervisor) Run() {
	webapi.GlobalProxy.Enqueue(webapi.Started)
	meter.SupSideStart()

	// 起一个听的循环
	go d.doAccept()

	d.Wait()

	// 无脑发送全部东西给主节点。
	d.cmt.MsgSendingControl()
	supervisor_log.DebugLog.Println("afterMsgSendingControl")

	// 发完之后开始准备在恰当时机发送Stop信息。
	for !d.Ss.GapEnough() { // wait all txs to be handled
		time.Sleep(time.Second)
	}
	// send stop message
	d.sl.Slog.Println("Supervisor: now sending stop message to all nodes")

	for nid := uint64(0); nid < uint64(params.NodeNum); nid++ {
		supervisor_log.DebugLog.Printf("Sending a %v: %v\n", pbft.CStop, string([]byte("this is a stop message~")))
		pbft.MergeAndSend(pbft.CStop, []byte("this is a stop message~"), d.IpNodeTable[0][nid], supervisor_log.DebugLog)
	}

	d.sl.Slog.Println("Supervisor: now Closing. Now Generate Metrics Outputs.")

	d.generateOutputAndCleanUp()
}

// handle pbft. only one message to be handled now
func (d *Supervisor) dispatchMessage(msg []byte) {
	msgType, content := pbft.SplitMessage(msg)
	if len(content) > 2000 {
		supervisor_log.DebugLog.Printf("Received a %v: %v\n", msgType, string(content[:2000]))
	} else {
		supervisor_log.DebugLog.Printf("Received a %v: %v\n", msgType, string(content))
	}
	switch msgType {
	case pbft.CKeepAlive: // 用于确认主节点的启动情况
		m := new(pbft.KeepAliveMsg)
		err := json.Unmarshal(content, m)
		if err != nil {
			log.Panic()
		}
		d.handleKeepAliveMsg(m)

	case pbft.CBlockInfo: // 统计区块相关指标
		m := new(pbft.BlockInfoMsg)
		err := json.Unmarshal(content, m)
		if err != nil {
			log.Panic()
		}
		d.handleBlockInfoMsg(m)
		si := sig.GetSignalByName[*pbft.BlockInfoMsg]("OnBimReached")
		si.Emit(m)

	case pbft.CBooking: // 统计节点运行状态相关指标
		m := new(pbft.BookingMsg)
		err := json.Unmarshal(content, m)
		if err != nil {
			log.Panic()
		}
		d.handleBookingMsg(m)

	default:
		panic("Message Unsupported")
	}
}

func (d *Supervisor) startSession(con net.Conn) {
	defer con.Close()
	clientReader := bufio.NewReader(con)
	for {
		clientRequest, err := clientReader.ReadBytes('\n')
		switch err {
		case nil:
			d.tcpLock.Lock()
			d.dispatchMessage(clientRequest)
			d.tcpLock.Unlock()
		case io.EOF:
			log.Println("client closed the connection by terminating the process")
			return
		default:
			log.Printf("error: %v\n", err)
			return
		}
	}
}

func (d *Supervisor) doAccept() {
	ch := network.Tcp.Serve(params.SupervisorEndpoint)
	for {
		clientRequest, ok := <-ch
		if !ok {
			d.sl.Slog.Println("Sup, the Tcp channel is closed")
			return
		}
		log.Printf("Receiving %v", clientRequest)
		d.dispatchMessage(clientRequest)
	}
}

// close Supervisor, and record the data in .csv file
func (d *Supervisor) generateOutputAndCleanUp() {
	d.sl.Slog.Println("Closing...")

	d.sl.Slog.Println("Now waiting for Other Node Bookings and result")

	result := <-d.result
	d.sl.Slog.Println("result generated")

	webapi.GlobalProxy.Enqueue(webapi.Completed(d.pbftItems, result))

	// 保证路径存在
	err := os.MkdirAll(params.DataWritePath, 0755)
	if err != nil {
		panic(err)
	}

	// pbft_tx.json
	pbftTxJsonBytes, err := json.Marshal(d.pbftItems)
	if err != nil {
		panic(err)
	}
	fPbpfTx, err := os.OpenFile(path.Join(params.DataWritePath, "pbft_tx.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	_, err = fPbpfTx.Write(pbftTxJsonBytes)
	if err != nil {
		panic(err)
	}
	err = fPbpfTx.Close()
	if err != nil {
		panic(err)
	}

	// metrics_result.json
	metricsResultJsonBytes, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	fMetricsResult, err := os.OpenFile(path.Join(params.DataWritePath, "metrics_result.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	_, err = fMetricsResult.Write(metricsResultJsonBytes)
	if err != nil {
		panic(err)
	}
	err = fMetricsResult.Close()
	if err != nil {
		panic(err)
	}

	network.Tcp.Close()
	webapi.GlobalProxy.Enqueue(webapi.Bye)
}
