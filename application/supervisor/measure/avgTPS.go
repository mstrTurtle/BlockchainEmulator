package measure

import (
	"fmt"
	"time"

	"github.com/Aj002Th/BlockchainEmulator/application/supervisor/metrics"
	"github.com/Aj002Th/BlockchainEmulator/consensus/pbft"
)

// to test average TPS in this system
type TestModule_avgTPS_Pbft struct {
	epochID      int
	excutedTxNum []float64       // record how many excuted txs in a epoch, maybe the cross shard tx will be calculated as a 0.5 tx
	pbftTx       map[string]bool // record whether a pbftTx or not
	startTime    []time.Time     // record when the epoch starts
	endTime      []time.Time     // record when the epoch ends
}

func NewTestModule_avgTPS_Pbft() *TestModule_avgTPS_Pbft {
	return &TestModule_avgTPS_Pbft{
		epochID:      -1,
		excutedTxNum: make([]float64, 0),
		startTime:    make([]time.Time, 0),
		endTime:      make([]time.Time, 0),
		pbftTx:       make(map[string]bool),
	}
}

func (tat *TestModule_avgTPS_Pbft) OutputMetricName() string {
	return "Average_TPS"
}

// add the number of excuted txs, and change the time records
func (tat *TestModule_avgTPS_Pbft) UpdateMeasureRecord(b *pbft.BlockInfoMsg) {
	if b.BlockBodyLength == 0 { // empty block
		return
	}

	epochid := b.Epoch
	earliestTime := b.ProposeTime
	latestTime := b.CommitTime

	// extend
	for tat.epochID < epochid {
		tat.excutedTxNum = append(tat.excutedTxNum, 0)
		tat.startTime = append(tat.startTime, time.Time{})
		tat.endTime = append(tat.endTime, time.Time{})
		tat.epochID++
	}
	// modify the local epoch
	for _, tx := range b.Pbft1Txs {
		tat.pbftTx[string(tx.Hash)] = true
	}
	tat.excutedTxNum[epochid] += float64(b.Pbft1TxNum) / 2
	for _, tx := range b.ExcutedTxs {
		if _, ok := tat.pbftTx[string(tx.Hash)]; ok {
			tat.excutedTxNum[epochid] += 0.5
		} else {
			tat.excutedTxNum[epochid] += 1
		}
	}
	if tat.startTime[epochid].IsZero() || tat.startTime[epochid].After(earliestTime) {
		tat.startTime[epochid] = earliestTime
	}
	if tat.endTime[epochid].IsZero() || latestTime.After(tat.endTime[epochid]) {
		tat.endTime[epochid] = latestTime
	}
}

// output the average TPS
func (tat *TestModule_avgTPS_Pbft) OutputRecord() (perEpochTPS []float64, totalTPS float64) {
	perEpochTPS = make([]float64, tat.epochID+1)
	totalTxNum := 0.0
	eTime := time.Now()
	lTime := time.Time{}
	for eid, exTxNum := range tat.excutedTxNum {
		timeGap := tat.endTime[eid].Sub(tat.startTime[eid]).Seconds()
		perEpochTPS[eid] = exTxNum / timeGap
		totalTxNum += exTxNum
		if eTime.After(tat.startTime[eid]) {
			eTime = tat.startTime[eid]
		}
		if tat.endTime[eid].After(lTime) {
			lTime = tat.endTime[eid]
		}
	}
	totalTPS = totalTxNum / (lTime.Sub(eTime).Seconds())
	return
}

func fmtTxPerSec(v float64) string {
	return fmt.Sprintf("%.2f txs/s", v)
}

func (tat *TestModule_avgTPS_Pbft) GetDesc() metrics.Desc {

	b := metrics.NewDescBuilder("交易共识频率(AverageTPS)", "平均每秒产生的交易，衡量交易的次数。单位为 交易/秒")

	var perEpochTPS []float64
	var totalTPS float64
	perEpochTPS = make([]float64, tat.epochID+1)
	totalTxNum := 0.0
	eTime := time.Now()
	lTime := time.Time{}
	for eid, exTxNum := range tat.excutedTxNum {
		timeGap := tat.endTime[eid].Sub(tat.startTime[eid]).Seconds()
		b.AddElem(fmt.Sprintf("第%v批次 交易共识频率", eid+1), fmt.Sprintf("第%v批次过程中产生交易的交易共识频率", eid+1), fmtTxPerSec(exTxNum/timeGap))
		perEpochTPS[eid] = exTxNum / timeGap
		totalTxNum += exTxNum
		if eTime.After(tat.startTime[eid]) {
			eTime = tat.startTime[eid]
		}
		if tat.endTime[eid].After(lTime) {
			lTime = tat.endTime[eid]
		}
	}
	totalTPS = totalTxNum / (lTime.Sub(eTime).Seconds())
	b.AddElem("总交易共识频率", "整个过程中平均每秒产生的交易", fmtTxPerSec(totalTPS))
	return b.GetDesc()

}
