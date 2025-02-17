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

package replica

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/lindb/config"
	"github.com/lindb/lindb/models"
	"github.com/lindb/lindb/pkg/timeutil"
	protoMetricsV1 "github.com/lindb/lindb/proto/gen/v1/metrics"
	"github.com/lindb/lindb/rpc"
	"github.com/lindb/lindb/series/metric"
)

func TestChannel_Write(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()

	stream := rpc.NewMockWriteStream(ctrl)
	stream.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	stream.EXPECT().Close().AnyTimes()

	ctx, cancel := context.WithCancel(context.TODO())
	ch := newFamilyChannel(ctx, config.GlobalBrokerConfig().Write, "database", 1, 12, nil, models.ShardState{}, nil)
	ch1 := ch.(*familyChannel)
	ch1.lock4write.Lock()
	ch1.newWriteStreamFn = func(ctx context.Context, target models.Node, database string,
		shardState *models.ShardState, familyTime int64, fct rpc.ClientStreamFactory) (rpc.WriteStream, error) {
		return stream, nil
	}
	ch1.lock4write.Unlock()

	converter := metric.NewProtoConverter()
	var brokerRow metric.BrokerRow
	assert.NoError(t, converter.ConvertTo(&protoMetricsV1.Metric{
		Name:      "cpu",
		Timestamp: timeutil.Now(),
		SimpleFields: []*protoMetricsV1.SimpleField{
			{Name: "f1", Type: protoMetricsV1.SimpleFieldType_DELTA_SUM, Value: 1}},
	}, &brokerRow))
	err := ch.Write(context.TODO(), []metric.BrokerRow{brokerRow})
	assert.NoError(t, err)
	err = ch.Write(context.TODO(), []metric.BrokerRow{brokerRow})
	assert.NoError(t, err)

	cancel()
	time.Sleep(time.Millisecond * 600)

	ch = newFamilyChannel(ctx, config.GlobalBrokerConfig().Write, "database", 1, 12, nil, models.ShardState{}, nil)
	ch1 = ch.(*familyChannel)
	ch1.lock4write.Lock()
	ch1.newWriteStreamFn = func(ctx context.Context, target models.Node, database string,
		shardState *models.ShardState, familyTime int64, fct rpc.ClientStreamFactory) (rpc.WriteStream, error) {
		return stream, nil
	}
	ch1.lock4write.Unlock()
	time.Sleep(time.Millisecond * 600) // wait task finish
	// ignore data, after closed
	chunk := NewMockChunk(ctrl)
	ch1.chunk = chunk
	// make sure chan is full
	var data = compressedChunk([]byte{1, 2})
	ch1.ch <- &data
	ch1.ch <- &data
	chunk.EXPECT().Write(gomock.Any())
	chunk.EXPECT().IsFull().Return(true)
	data2 := compressedChunk([]byte{1, 2, 3})
	chunk.EXPECT().Compress().Return(&data2, nil)
	err = ch.Write(context.TODO(), []metric.BrokerRow{brokerRow})
	assert.Error(t, err)
	time.Sleep(time.Millisecond * 500)
}

func TestChannel_checkFlush(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()

	stream := rpc.NewMockWriteStream(ctrl)
	stream.EXPECT().Close().Return(nil).AnyTimes()
	stream.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.TODO())
	ch := newFamilyChannel(ctx, config.GlobalBrokerConfig().Write, "database", 1, 12, nil, models.ShardState{}, nil)
	ch1 := ch.(*familyChannel)
	ch1.lock4write.Lock()
	ch1.newWriteStreamFn = func(ctx context.Context, target models.Node, database string,
		shardState *models.ShardState, familyTime int64, fct rpc.ClientStreamFactory) (rpc.WriteStream, error) {
		return stream, nil
	}
	ch1.lock4write.Unlock()

	converter := metric.NewProtoConverter()
	var brokerRow metric.BrokerRow
	assert.NoError(t, converter.ConvertTo(&protoMetricsV1.Metric{
		Name:      "cpu",
		Timestamp: timeutil.Now(),
		SimpleFields: []*protoMetricsV1.SimpleField{
			{Name: "f1", Type: protoMetricsV1.SimpleFieldType_DELTA_SUM, Value: 1}},
	}, &brokerRow))

	err := ch.Write(context.TODO(), []metric.BrokerRow{brokerRow})
	assert.NoError(t, err)

	time.Sleep(time.Second)
	cancel()
	time.Sleep(300 * time.Millisecond)
}

func TestChannel_write_pending_before_close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	stream := rpc.NewMockWriteStream(ctrl)
	stream.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	stream.EXPECT().Close().Return(nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := newFamilyChannel(ctx, config.GlobalBrokerConfig().Write, "database", 1, 12, nil, models.ShardState{}, nil)
	ch1 := ch.(*familyChannel)
	ch1.lock4write.Lock()
	ch1.newWriteStreamFn = func(ctx context.Context, target models.Node, database string,
		shardState *models.ShardState, familyTime int64, fct rpc.ClientStreamFactory) (rpc.WriteStream, error) {
		return stream, nil
	}
	ch1.lock4write.Unlock()

	converter := metric.NewProtoConverter()
	var brokerRow metric.BrokerRow
	assert.NoError(t, converter.ConvertTo(&protoMetricsV1.Metric{
		Name:      "cpu",
		Timestamp: timeutil.Now(),
		SimpleFields: []*protoMetricsV1.SimpleField{
			{Name: "f1", Type: protoMetricsV1.SimpleFieldType_DELTA_SUM, Value: 1}},
	}, &brokerRow))

	err := ch.Write(context.TODO(), []metric.BrokerRow{brokerRow})
	assert.NoError(t, err)

	var data = compressedChunk([]byte{1, 2, 3})
	ch1.ch <- &data
	ch1.writePendingBeforeClose()
}

func TestChannel_chunk_marshal_err(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	stream := rpc.NewMockWriteStream(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := newFamilyChannel(ctx, config.GlobalBrokerConfig().Write, "database", 1, 12, nil, models.ShardState{}, nil)
	ch1 := ch.(*familyChannel)
	ch1.lock4write.Lock()
	ch1.newWriteStreamFn = func(ctx context.Context, target models.Node, database string,
		shardState *models.ShardState, familyTime int64, fct rpc.ClientStreamFactory) (rpc.WriteStream, error) {
		return stream, nil
	}
	ch1.lock4write.Unlock()

	chunk := NewMockChunk(ctrl)
	ch1.chunk = chunk

	converter := metric.NewProtoConverter()
	var brokerRow metric.BrokerRow
	assert.NoError(t, converter.ConvertTo(&protoMetricsV1.Metric{
		Name:      "cpu",
		Timestamp: timeutil.Now(),
		SimpleFields: []*protoMetricsV1.SimpleField{
			{Name: "f1", Type: protoMetricsV1.SimpleFieldType_DELTA_SUM, Value: 1}},
	}, &brokerRow))

	chunk.EXPECT().Write(gomock.Any())
	chunk.EXPECT().IsFull().Return(true)
	chunk.EXPECT().Compress().Return(nil, fmt.Errorf("err"))
	err := ch.Write(context.TODO(), []metric.BrokerRow{brokerRow})
	assert.Error(t, err)

	chunk.EXPECT().Compress().Return(nil, fmt.Errorf("err"))
	ch1.flushChunk()
	chunk.EXPECT().Compress().Return(nil, nil)
	ch1.flushChunk()
}
