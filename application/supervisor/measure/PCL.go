package measure

import (
	"fmt"

	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/metrics"
	"github.com/Aj002Th/BlockchainEmulator/consensus/pbft"
)

// PCL to test average Transaction_Confirm_Latency (TCL)  in this system
type PCL struct {
	epochID           int       // 其实这应该叫epochCnt，随着bim到来，这个值是计数递增的。
	totTxLatencyEpoch []float64 // record the Transaction_Confirm_Latency in each epoch
	txNum             []float64 // record the txNumber in each epoch
}

func NewPCL() *PCL {
	return &PCL{
		epochID:           -1,
		totTxLatencyEpoch: make([]float64, 0),
		txNum:             make([]float64, 0),
	}
}

func (tml *PCL) OutputMetricName() string {
	return "Transaction_Confirm_Latency"
}

// UpdateMeasureRecord modified latency
func (tml *PCL) UpdateMeasureRecord(b *pbft.BlockInfoMsg) {
	if b.BlockBodyLength == 0 { // empty block
		return
	}

	epochid := b.Epoch
	txs := b.ExcutedTxs
	mTime := b.CommitTime

	// extend
	for tml.epochID < epochid {
		tml.txNum = append(tml.txNum, 0)
		tml.totTxLatencyEpoch = append(tml.totTxLatencyEpoch, 0)
		tml.epochID++
	}

	for _, tx := range txs {
		if !tx.Time.IsZero() {
			tml.totTxLatencyEpoch[epochid] += mTime.Sub(b.ProposeTime).Seconds()
			tml.txNum[epochid]++
		}
	}
}

func (tml *PCL) OutputRecord() (perEpochLatency []float64, totLatency float64) {
	perEpochLatency = make([]float64, 0)
	latencySum := 0.0
	totTxNum := 0.0
	for eid, totLatency := range tml.totTxLatencyEpoch {
		perEpochLatency = append(perEpochLatency, totLatency/tml.txNum[eid])
		latencySum += totLatency
		totTxNum += tml.txNum[eid]
	}
	totLatency = latencySum / totTxNum
	return
}

func (tml *PCL) GetDesc() metrics.Desc {
	b := metrics.NewDescBuilder("Prepare-Commit时延(PCL)", "交易从Propose阶段到Commit阶段的耗时统计")

	var perEpochLatency []float64
	var totLatency float64
	perEpochLatency = make([]float64, 0)
	latencySum := 0.0
	totTxNum := 0.0
	for eid, totLatency := range tml.totTxLatencyEpoch {
		b.AddElem(fmt.Sprintf("第%v批次 Prepare-Commit时延", eid+1), "本批次交易从Propose阶段到Commit阶段的耗时统计", fmt.Sprintf("%.6f s", totLatency/tml.txNum[eid]))
		perEpochLatency = append(perEpochLatency, totLatency/tml.txNum[eid])
		latencySum += totLatency
		totTxNum += tml.txNum[eid]
	}
	totLatency = latencySum / totTxNum

	b.AddElem("总计Prepare-Commit时延", "整个运行过程中交易从Propose阶段到Commit阶段的耗时统计", fmt.Sprintf("%.6f s", totLatency))
	return b.GetDesc()
}
