package meter

import (
	"fmt"

	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/metrics"
	"github.com/Aj002Th/BlockchainEmulator/consensus/pbft"
	"github.com/Aj002Th/BlockchainEmulator/params"
	"github.com/Aj002Th/BlockchainEmulator/signal"
	"github.com/chebyrash/promise"
)

type Void = struct{}
type Booking = pbft.Booking

func SupSideStart() {
	signal.GetSignalByName[Void]("OnSupStart").Connect(func(Void) {
		// Sup只需启动Commit相关，因为这些计算比较重。
		StartCommitRelate()
	})
}

var SupOnGathered signal.Signal[metrics.Desc]

func GetNReturnAsync() *promise.Promise[[]Booking] {
	sig := signal.GetSignalByName[Booking]("OnBookingReach")
	cnt := 0
	bks := make([]Booking, 0)
	p := promise.New(func(resolve func([]Booking), reject func(error)) {
		sig.Connect(func(bk Booking) {
			cnt++
			bks = append(bks, bk)
			if cnt == params.NodeNum {
				resolve(bks)
			}
		})
	})
	return p
}

func GetResult(ws *[]Booking) []metrics.Desc { // 每一个度量，作为一棵树，都是一个Desc。现在需要一系列Desc。
	// pws := GetNReturnAsync()
	// ws, _ := pws.Await(context.Background()) // 暂时没err，不用管err
	var ds = make([]metrics.Desc, 0)

	// 统计TxCount和BlockNum和运行时间
	tx := metrics.NewDescBuilder("CPU时间", "交易计数，是指对交易的计数。")
	bc := metrics.NewDescBuilder("内存测量", "交易计数，是指对交易的计数。")
	dur := metrics.NewDescBuilder("时间", "")
	net := metrics.NewDescBuilder("网络", "")
	var sumC float64 = 0
	var sumBc uint64 = 0
	var sumDur uint64 = 0
	var sumUp, sumDown int = 0, 0
	for _, w := range *ws {
		nn := w.NodeId
		c := w.AvgCpuTime
		b := w.DiskMetric
		t := w.TotalTime
		tx.AddElem(fmt.Sprintf("节点%v CPU时间", nn), "", c)
		bc.AddElem(fmt.Sprintf("节点%v 内存测量", nn), "", b)
		dur.AddElem(fmt.Sprintf("节点%v 时间", nn), "", t)
		net.AddElem(fmt.Sprintf("节点%v 上传", nn), "", w.TotalUpload)
		net.AddElem(fmt.Sprintf("节点%v 下载", nn), "", w.TotalDownload)
		sumC += c
		sumBc += b
		sumDur += t
		sumUp += w.TotalUpload
		sumDown += w.TotalDownload
	}
	tx.AddElem("平均计数", "", sumC/float64(params.NodeNum))
	bc.AddElem("平均计数", "", sumBc/uint64(params.NodeNum))
	dur.AddElem("平均运行时间", "", sumDur/uint64(params.NodeNum))
	net.AddElem("平均上传流量", "", sumUp/params.NodeNum)
	net.AddElem("平均下载流量", "", sumDown/params.NodeNum)
	ds = append(ds, tx.GetDesc())
	ds = append(ds, bc.GetDesc())
	ds = append(ds, dur.GetDesc())
	ds = append(ds, net.GetDesc())

	// 那堆传统模块
	ds = append(ds, m1.GetDesc())
	ds = append(ds, m2.GetDesc())
	ds = append(ds, m3.GetDesc())
	ds = append(ds, m4.GetDesc())
	ds = append(ds, m5.GetDesc())
	ds = append(ds, m6.GetDesc())
	return ds
}