package meter

import (
	"context"

	"github.com/Aj002Th/BlockchainEmulator/application/comm"
	"github.com/Aj002Th/BlockchainEmulator/signal"
	"github.com/chebyrash/promise"
)

func SubscribeSignalOnce[DATA any](name string) *promise.Promise[DATA] {
	sig := signal.GetSignalByName[DATA](name)
	return promise.New[DATA](func(resolve func(DATA), reject func(error)) {
		var cb func(data DATA)
		cb = func(data DATA) {
			// 分门别类把它们传回去。
			resolve(data)
			sig.Disconnect(cb)
		}
		sig.Connect(cb)
	})
}

func StartNodeSideGather() {
	p := SubscribeSignalOnce[Void]("OnEmulatorStop")
	promise.Then(p, context.Background(), OnEmulatorStop)

}

type Booking struct {
	AvgCpuTime    float64 `json:"avgCpuTime"`
	DiskMetric    uint64  `json:"disk"`
	TxCount       uint64  `json:"txc"`
	BlockCount    uint64  `json:"bc"`
	TotalUpload   int     `json:"tu"`
	TotalDownload int     `json:"td"`
	TotalTime     uint64  `json:"tm"`
	NodeId        int     `json:"nodeid"`
}

func OnEmulatorStop(Void) (Void, error) {
	// Procs相关
	comm.Dial()
	b := Booking{AvgCpuTime: avgCpuTime, DiskMetric: diskMetric, TxCount: txCount, BlockCount: BlockCount, TotalUpload: TotalUpload, TotalDownload: TotalDownload}
	w := comm.Wrapper{MsgType: "Bookeeping", Content: b}
	comm.Send(w)
	return Void{}, nil
}
