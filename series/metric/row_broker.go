// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metric

import (
	"io"
	"sort"
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/lithammer/go-jump-consistent-hash"

	"github.com/lindb/lindb/models"
	"github.com/lindb/lindb/pkg/encoding"
	"github.com/lindb/lindb/pkg/fasttime"
	"github.com/lindb/lindb/pkg/timeutil"
	"github.com/lindb/lindb/proto/gen/v1/flatMetricsV1"
)

type BrokerRow struct {
	m      flatMetricsV1.Metric
	buffer []byte

	ShardID models.ShardID
	// IsOutOfTimeRange marks if this row is out-of time-range
	// data is not accessible when its set to true
	IsOutOfTimeRange bool
}

// FromBlock resets buffer, unmarshal from a new block,
// make sure that metric and shard id will be overwritten manually
func (row *BrokerRow) FromBlock(block []byte) {
	row.buffer = encoding.MustCopy(row.buffer, block)
	size := flatbuffers.GetSizePrefix(row.buffer, 0)
	partition := row.buffer[flatbuffers.SizeUOffsetT : flatbuffers.SizeUOffsetT+size]
	row.m.Init(partition, flatbuffers.GetUOffsetT(partition))
}

func (row *BrokerRow) Metric() flatMetricsV1.Metric { return row.m }

func (row *BrokerRow) Size() int {
	if row.IsOutOfTimeRange {
		return 0
	}
	return len(row.buffer)
}

func (row *BrokerRow) WriteTo(writer io.Writer) (int, error) {
	if row.IsOutOfTimeRange {
		return 0, nil
	}
	return writer.Write(row.buffer)
}

var brokerBatchRowsPool sync.Pool

// BrokerBatchRows holds rows from ingestion
// row will be putted into buffer after validation and re-building
type BrokerBatchRows struct {
	rows     []BrokerRow
	rowCount int

	shardGroupIterator BrokerBatchShardIterator
}

func newBrokerBatchRows() *BrokerBatchRows {
	return &BrokerBatchRows{}
}

// NewBrokerBatchRows returns a new batch for decoding flat metrics.
func NewBrokerBatchRows() (batch *BrokerBatchRows) {
	item := brokerBatchRowsPool.Get()
	if item != nil {
		builder := item.(*BrokerBatchRows)
		builder.reset()
		return builder
	}
	return newBrokerBatchRows()
}

// Release releases rows context into sync.Pool
func (br *BrokerBatchRows) Release() { brokerBatchRowsPool.Put(br) }

func (br *BrokerBatchRows) reset() { br.rowCount = 0 }

func (br *BrokerBatchRows) Len() int { return br.rowCount }
func (br *BrokerBatchRows) Less(i, j int) bool {
	return br.rows[i].ShardID < br.rows[j].ShardID
}
func (br *BrokerBatchRows) Swap(i, j int)     { br.rows[i], br.rows[j] = br.rows[j], br.rows[i] }
func (br *BrokerBatchRows) Rows() []BrokerRow { return br.rows[:br.rowCount] }

// EvictOutOfTimeRange evicts and marks out-of-range metrics invalid
func (br *BrokerBatchRows) EvictOutOfTimeRange(behind, ahead int64) (evicted int) {
	// check metric timestamp if in acceptable time range
	now := fasttime.UnixMilliseconds()
	for idx := 0; idx < br.Len(); idx++ {
		if (behind > 0 && br.rows[idx].m.Timestamp() < now-behind) ||
			(ahead > 0 && br.rows[idx].m.Timestamp() > now+ahead) {
			br.rows[idx].IsOutOfTimeRange = true
			evicted++
		}
	}
	return evicted
}

func (br *BrokerBatchRows) TryAppend(appendFunc func(row *BrokerRow) error) error {
	if len(br.rows) <= br.rowCount {
		br.rows = append(br.rows, BrokerRow{})
	}
	if err := appendFunc(&br.rows[br.rowCount]); err != nil {
		return err
	}
	// decoded successfully, move to next row index
	br.rowCount++
	return nil
}

func (br *BrokerBatchRows) NewShardGroupIterator(numOfShards int32) *BrokerBatchShardIterator {
	for i := 0; i < br.Len(); i++ {
		br.rows[i].ShardID = models.ShardID(jump.Hash(br.rows[i].m.Hash(), numOfShards))
	}
	br.shardGroupIterator.batch = br
	br.shardGroupIterator.Reset()
	return &br.shardGroupIterator
}

// BrokerBatchShardIterator grouping broker rows with shard-id,
// rows will be batched inserted into shard-channel for replication
type BrokerBatchShardIterator struct {
	groupEnd     int            // group end index
	groupStart   int            // group start index
	groupShardID models.ShardID // group shard id

	batch *BrokerBatchRows

	familyIterator BrokerBatchShardFamilyIterator
}

// Reset re-sorts batch rows for batching inserting
func (itr *BrokerBatchShardIterator) Reset() {
	sort.Sort(itr.batch)
	itr.groupStart = 0
	itr.groupEnd = 0
	itr.groupShardID = models.ShardID(-1)
}

func (itr *BrokerBatchShardIterator) HasRowsForNextShard() bool {
	if itr.groupEnd >= itr.batch.Len() || itr.groupStart > itr.groupEnd {
		return false
	}
	itr.groupShardID = itr.batch.rows[itr.groupEnd].ShardID
	itr.groupStart = itr.groupEnd

	for itr.groupEnd < itr.batch.Len() {
		if !(itr.batch.rows[itr.groupEnd].ShardID == itr.groupShardID) {
			break
		}
		itr.groupEnd++
	}
	return itr.groupStart < itr.groupEnd
}

func (itr *BrokerBatchShardIterator) FamilyRowsForNextShard(
	interval timeutil.Interval,
) (
	shardID models.ShardID,
	familyIterator *BrokerBatchShardFamilyIterator,
) {
	itr.familyIterator.reset(
		itr.batch.rows[itr.groupStart:itr.groupEnd],
		interval,
	)
	return itr.groupShardID, &itr.familyIterator
}

// BrokerBatchShardFamilyIterator grouping broker rows with families
// rows will be batched inserted into shard-channel for replication
type BrokerBatchShardFamilyIterator struct {
	groupEnd        int
	groupStart      int
	groupFamilyTime int64 // group family time

	sameFamily bool

	rows familySortedRows

	intervalCalc timeutil.IntervalCalculator
}

func (itr *BrokerBatchShardFamilyIterator) reset(
	rows []BrokerRow,
	interval timeutil.Interval,
) {
	itr.groupEnd = 0
	itr.groupStart = 0
	itr.rows = rows
	itr.intervalCalc = interval.Calculator()
	itr.groupFamilyTime = 0
	itr.rows = rows
	// fast path, all rows are same family
	if itr.sameFamily = itr.isSameFamily(); itr.sameFamily {
		return
	}
	sort.Sort(itr.rows)
}

func (itr *BrokerBatchShardFamilyIterator) isSameFamily() bool {
	if len(itr.rows) == 0 {
		return true
	}
	firstTimestamp := itr.rows[0].m.Timestamp()
	itr.groupFamilyTime = itr.familyTimeOfTimestamp(firstTimestamp)
	timeRange := itr.timeRangeOfTimestamp(firstTimestamp)
	for i := 1; i < len(itr.rows); i++ {
		if !timeRange.Contains(itr.rows[i].m.Timestamp()) {
			return false
		}
	}
	return true
}

func (itr *BrokerBatchShardFamilyIterator) HasNextFamily() bool {
	if itr.groupEnd >= len(itr.rows) || itr.groupStart > itr.groupEnd {
		return false
	}
	if itr.sameFamily {
		itr.groupEnd = len(itr.rows)
		itr.groupStart = 0
		return true
	}

	firstTimestamp := itr.rows[itr.groupEnd].m.Timestamp()
	timeRange := itr.timeRangeOfTimestamp(firstTimestamp)
	itr.groupStart = itr.groupEnd
	itr.groupFamilyTime = itr.familyTimeOfTimestamp(firstTimestamp)

	for itr.groupEnd < len(itr.rows) {
		if !timeRange.Contains(itr.rows[itr.groupEnd].m.Timestamp()) {
			break
		}
		itr.groupEnd++
	}
	return itr.groupStart < itr.groupEnd
}

func (itr *BrokerBatchShardFamilyIterator) NextFamily() (familyTime int64, rows []BrokerRow) {
	return itr.groupFamilyTime, itr.rows[itr.groupStart:itr.groupEnd]
}

func (itr *BrokerBatchShardFamilyIterator) familyTimeOfTimestamp(timestamp int64) int64 {
	segmentTime := itr.intervalCalc.CalcSegmentTime(timestamp)
	family := itr.intervalCalc.CalcFamily(timestamp, segmentTime)
	return itr.intervalCalc.CalcFamilyStartTime(segmentTime, family)
}

func (itr *BrokerBatchShardFamilyIterator) timeRangeOfTimestamp(timestamp int64) timeutil.TimeRange {
	segmentTime := itr.intervalCalc.CalcSegmentTime(timestamp)
	family := itr.intervalCalc.CalcFamily(timestamp, segmentTime)
	familyStartTime := itr.intervalCalc.CalcFamilyStartTime(segmentTime, family)
	return timeutil.TimeRange{
		Start: familyStartTime,
		End:   itr.intervalCalc.CalcFamilyEndTime(familyStartTime),
	}
}

// sort rows by timestamp, so we will
type familySortedRows []BrokerRow

func (fr familySortedRows) Len() int           { return len(fr) }
func (fr familySortedRows) Less(i, j int) bool { return fr[i].m.Timestamp() < fr[j].m.Timestamp() }
func (fr familySortedRows) Swap(i, j int)      { fr[i], fr[j] = fr[j], fr[i] }
