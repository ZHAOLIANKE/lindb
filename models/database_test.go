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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lindb/lindb/pkg/option"
)

func TestNewShardAssignment(t *testing.T) {
	shardAssign := NewShardAssignment("test")
	shardAssign.AddReplica(1, 1)
	shardAssign.AddReplica(1, 2)
	shardAssign.AddReplica(2, 3)
	shardAssign.AddReplica(2, 5)
	assert.Equal(t, []NodeID{1, 2}, shardAssign.Shards[1].Replicas)
	assert.Equal(t, []NodeID{3, 5}, shardAssign.Shards[2].Replicas)
}

func TestDatabase_String(t *testing.T) {
	database := Database{
		Name:          "test",
		NumOfShard:    10,
		ReplicaFactor: 1,
		Option:        option.DatabaseOption{Interval: "10s"},
	}
	assert.Equal(t, "create database test with shard 10, replica 1, interval 10s", database.String())
}
