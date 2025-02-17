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

package constants

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound represents the data not found
	ErrNotFound = errors.New("not found")

	ErrTagFilterResultNotFound      = fmt.Errorf("tagFilter result %w", ErrNotFound)
	ErrTagValueFilterResultNotFound = fmt.Errorf("tagValueFitler result %w", ErrNotFound)

	ErrDatabaseNotFound        = fmt.Errorf("database %w", ErrNotFound)
	ErrShardNotFound           = fmt.Errorf("shard %w", ErrNotFound)
	ErrReplicaNotFound         = fmt.Errorf("replica %w", ErrNotFound)
	ErrNameSpaceBucketNotFound = fmt.Errorf("namespace bucket %w", ErrNotFound)
	ErrMetricIDNotFound        = fmt.Errorf("metricID %w", ErrNotFound)
	ErrMetricBucketNotFound    = fmt.Errorf("metric bucket %w", ErrNotFound)
	ErrHistogramFieldNotFound  = fmt.Errorf("histogram field %w", ErrNotFound)
	ErrTagKeyIDNotFound        = fmt.Errorf("tagKeyID %w", ErrNotFound)
	ErrTagKeyMetaNotFound      = fmt.Errorf("tagKeyMeta %w", ErrNotFound)
	ErrTagValueSeqNotFound     = fmt.Errorf("tagValueSeq %w", ErrNotFound)
	ErrTagValueIDNotFound      = fmt.Errorf("tagValueID %w", ErrNotFound)
	ErrFieldNotFound           = fmt.Errorf("field %w", ErrNotFound)
	ErrFieldBucketNotFound     = fmt.Errorf("field bucket %w", ErrNotFound)
	ErrSeriesIDNotFound        = fmt.Errorf("seriesID %w", ErrNotFound)
	ErrDataFamilyNotFound      = fmt.Errorf("data family %w", ErrNotFound)

	ErrMetricOutOfTimeRange = errors.New("metric's timestamp is out of timerange")

	// ErrDataFileCorruption represents data in tsdb's file is corrupted
	ErrDataFileCorruption = errors.New("data corruption")

	ErrInfluxLineTooLong = errors.New("influx line is too long")

	ErrBadEnrichTagQueryFormat = errors.New("enrich_tag has the wrong format")
	// ErrNoLiveReplica represents no live replica node for current shard.
	ErrNoLiveReplica = errors.New("no live replica for shard")
	// ErrNoLiveNode represents no live node for current cluster.
	ErrNoLiveNode = errors.New("no live node for cluster")
	// ErrNameEmpty represents name is empty.
	ErrNameEmpty = errors.New("name cannot be empty")
	// ErrNoStorageCluster represents storage cluster not exist.
	ErrNoStorageCluster = errors.New("storage cluster not exist")
	// ErrStatefulNodeExist represents stateful node already register.
	ErrStatefulNodeExist = errors.New("stateful node already register")
)
