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

	"github.com/lindb/lindb/coordinator/storage"
	"github.com/lindb/lindb/models"
	"github.com/lindb/lindb/pkg/queue"
	protoReplicaV1 "github.com/lindb/lindb/proto/gen/v1/replica"
	"github.com/lindb/lindb/rpc"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRemoteReplicator_IsReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	cliFct := rpc.NewMockClientStreamFactory(ctrl)
	stateMgr := storage.NewMockStateManager(ctrl)
	stateMgr.EXPECT().GetLiveNode(gomock.Any()).Return(models.StatefulNode{}, true).AnyTimes()
	replicaCli := protoReplicaV1.NewMockReplicaServiceClient(ctrl)
	q := queue.NewMockFanOut(ctrl)
	fq := queue.NewMockFanOutQueue(ctrl)
	q.EXPECT().Queue().Return(fq).AnyTimes()
	rc := &ReplicatorChannel{
		State: &models.ReplicaState{
			Database: "test",
			ShardID:  0,
			Leader:   1,
			Follower: 2,
		},
		Queue: q,
	}

	r := NewRemoteReplicator(context.TODO(), rc, stateMgr, cliFct)
	r1 := r.(*remoteReplicator)
	// case 1: replicator is ready
	r1.state = ReplicatorReadyState
	assert.True(t, r.IsReady())

	r1.state = ReplicatorInitState
	// case 2: create replica cli err
	cliFct.EXPECT().CreateReplicaServiceClient(gomock.Any()).Return(nil, fmt.Errorf("err"))
	assert.False(t, r.IsReady())

	cliFct.EXPECT().CreateReplicaServiceClient(gomock.Any()).Return(replicaCli, nil).AnyTimes()

	// case 3: get replica stream err
	replicaCli.EXPECT().Replica(gomock.Any()).Return(nil, fmt.Errorf("err"))
	assert.False(t, r.IsReady())

	replicaCli.EXPECT().Replica(gomock.Any()).Return(nil, nil).AnyTimes()

	// case 4: get remote replica ack err
	replicaCli.EXPECT().GetReplicaAckIndex(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
	assert.False(t, r.IsReady())
	// case 5: replica idx == current node
	q.EXPECT().HeadSeq().Return(int64(11))
	replicaCli.EXPECT().GetReplicaAckIndex(gomock.Any(), gomock.Any()).Return(&protoReplicaV1.GetReplicaAckIndexResponse{
		AckIndex: 10,
	}, nil)
	assert.True(t, r.IsReady())
	// case 6: remote replica ack index < current smallest ack, but reset remote replica index err
	r = NewRemoteReplicator(context.TODO(), rc, stateMgr, cliFct)
	fq.EXPECT().HeadSeq().Return(int64(10))
	q.EXPECT().HeadSeq().Return(int64(12))
	q.EXPECT().TailSeq().Return(int64(13))
	replicaCli.EXPECT().GetReplicaAckIndex(gomock.Any(), gomock.Any()).Return(&protoReplicaV1.GetReplicaAckIndexResponse{
		AckIndex: 10,
	}, nil)
	replicaCli.EXPECT().Reset(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
	assert.False(t, r.IsReady())
	// case 7: remote replica ack index < current smallest ack, reset success
	r = NewRemoteReplicator(context.TODO(), rc, stateMgr, cliFct)
	fq.EXPECT().HeadSeq().Return(int64(10))
	q.EXPECT().HeadSeq().Return(int64(12))
	q.EXPECT().TailSeq().Return(int64(13))
	replicaCli.EXPECT().GetReplicaAckIndex(gomock.Any(), gomock.Any()).Return(&protoReplicaV1.GetReplicaAckIndexResponse{
		AckIndex: 10,
	}, nil)
	replicaCli.EXPECT().Reset(gomock.Any(), gomock.Any()).Return(nil, nil)
	q.EXPECT().SetHeadSeq(int64(11))
	assert.True(t, r.IsReady())
	// case 8: remote replica ack index > current append index, maybe leader lost data.
	r = NewRemoteReplicator(context.TODO(), rc, stateMgr, cliFct)
	fq.EXPECT().HeadSeq().Return(int64(5))
	q.EXPECT().HeadSeq().Return(int64(12))
	q.EXPECT().TailSeq().Return(int64(9))
	replicaCli.EXPECT().GetReplicaAckIndex(gomock.Any(), gomock.Any()).Return(&protoReplicaV1.GetReplicaAckIndexResponse{
		AckIndex: 10,
	}, nil)
	fq.EXPECT().SetAppendSeq(int64(11))
	q.EXPECT().SetHeadSeq(int64(11)).Return(fmt.Errorf("err"))
	assert.True(t, r.IsReady())
}

func TestRemoteReplicator_Replica(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	cliFct := rpc.NewMockClientStreamFactory(ctrl)
	stateMgr := storage.NewMockStateManager(ctrl)
	q := queue.NewMockFanOut(ctrl)
	rc := &ReplicatorChannel{
		State: &models.ReplicaState{
			Database: "test",
			ShardID:  0,
			Leader:   1,
			Follower: 2,
		},
		Queue: q,
	}

	r := NewRemoteReplicator(context.TODO(), rc, stateMgr, cliFct)
	r1 := r.(*remoteReplicator)
	cli := protoReplicaV1.NewMockReplicaService_ReplicaClient(ctrl)
	r1.replicaStream = cli

	cli.EXPECT().Send(gomock.Any()).Return(fmt.Errorf("err"))
	r.Replica(1, []byte{})

	cli.EXPECT().Send(gomock.Any()).Return(nil)
	cli.EXPECT().Recv().Return(nil, fmt.Errorf("err"))
	r.Replica(1, []byte{})

	cli.EXPECT().Send(gomock.Any()).Return(nil)
	cli.EXPECT().Recv().Return(&protoReplicaV1.ReplicaResponse{
		AckIndex: 1,
	}, nil)
	q.EXPECT().Ack(int64(1))
	r.Replica(1, []byte{})
}
