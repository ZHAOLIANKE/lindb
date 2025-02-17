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

package master

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/lindb/config"
	"github.com/lindb/lindb/coordinator/discovery"
	"github.com/lindb/lindb/coordinator/task"
	"github.com/lindb/lindb/models"
	"github.com/lindb/lindb/pkg/encoding"
	"github.com/lindb/lindb/pkg/option"
	"github.com/lindb/lindb/pkg/state"
)

func TestStateManager_Close(t *testing.T) {
	mgr := NewStateManager(context.TODO(), nil, nil, nil)
	fct := &StateMachineFactory{}
	mgr.SetStateMachineFactory(fct)
	assert.Equal(t, fct, mgr.GetStateMachineFactory())

	mgr.Close()
}

func TestStateManager_Handle_Event_Panic(t *testing.T) {
	mgr := NewStateManager(context.TODO(), nil, nil, nil)
	// case 1: panic
	mgr.EmitEvent(&discovery.Event{
		Type: discovery.DatabaseConfigDeletion,
		Key:  "/shard/assign/test",
	})
	time.Sleep(100 * time.Millisecond)
	mgr.Close()
}

func TestStateManager_NotRunning(t *testing.T) {
	mgr := NewStateManager(context.TODO(), nil, nil, nil)
	mgr1 := mgr.(*stateManager)
	mgr1.running.Store(false)
	// case 1: panic
	mgr.EmitEvent(&discovery.Event{
		Type: discovery.DatabaseConfigDeletion,
		Key:  "/shard/assign/test",
	})
	time.Sleep(100 * time.Millisecond)
	mgr.Close()
}

func TestStateManager_StorageCfg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mgr := NewStateManager(context.TODO(), nil, nil, nil)
	mgr1 := mgr.(*stateManager)
	//mgr1 := mgr.(*stateManager)
	// case 1: unmarshal cfg err
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.StorageConfigChanged,
		Key:   "/storage/test",
		Value: []byte("value"),
	})
	// case 2: storage name is empty
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.StorageConfigChanged,
		Key:   "/storage/test",
		Value: encoding.JSONMarshal(&config.StorageCluster{}),
	})
	// case 3: new storage cluster err
	mgr1.mutex.Lock()
	mgr1.newStorageClusterFn = func(ctx context.Context, cfg config.StorageCluster,
		stateMgr StateManager, repoFactory state.RepositoryFactory,
		controllerFactory task.ControllerFactory) (cluster StorageCluster, err error) {
		return nil, fmt.Errorf("err")
	}
	mgr1.mutex.Unlock()
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.StorageConfigChanged,
		Key:   "/storage/test",
		Value: encoding.JSONMarshal(&config.StorageCluster{Name: "test"}),
	})
	time.Sleep(100 * time.Millisecond)
	// case 4: start storage err
	storage1 := NewMockStorageCluster(ctrl)
	mgr1.mutex.Lock()
	mgr1.newStorageClusterFn = func(ctx context.Context, cfg config.StorageCluster,
		stateMgr StateManager, repoFactory state.RepositoryFactory,
		controllerFactory task.ControllerFactory) (cluster StorageCluster, err error) {
		return storage1, nil
	}
	mgr1.mutex.Unlock()
	storage1.EXPECT().Start().Return(fmt.Errorf("err"))
	storage1.EXPECT().Close()
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.StorageConfigChanged,
		Key:   "/storage/test",
		Value: encoding.JSONMarshal(&config.StorageCluster{Name: "test"}),
	})
	time.Sleep(100 * time.Millisecond)

	// case 5: start storage ok
	storage1.EXPECT().Start().Return(nil)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.StorageConfigChanged,
		Key:   "/storage/test",
		Value: encoding.JSONMarshal(&config.StorageCluster{Name: "test"}),
	})
	// case 6: remove not exist storage
	mgr.EmitEvent(&discovery.Event{
		Type: discovery.StorageDeletion,
		Key:  "/storage/test2",
	})
	time.Sleep(100 * time.Millisecond)
	storage := mgr.GetStorageCluster("test")
	assert.NotNil(t, storage)
	// case 7: remove storage
	storage1.EXPECT().Close()
	mgr.EmitEvent(&discovery.Event{
		Type: discovery.StorageDeletion,
		Key:  "/storage/test",
	})
	time.Sleep(100 * time.Millisecond)
	storage = mgr.GetStorageCluster("test")
	assert.Nil(t, storage)

	mgr.Close()
}

func TestStateManager_DatabaseCfg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	repo := state.NewMockRepository(ctrl)
	mgr := NewStateManager(context.TODO(), repo, nil, nil)
	mgr1 := mgr.(*stateManager)
	storage1 := NewMockStorageCluster(ctrl)
	mgr1.mutex.Lock()
	mgr1.newStorageClusterFn = func(ctx context.Context, cfg config.StorageCluster,
		stateMgr StateManager, repoFactory state.RepositoryFactory,
		controllerFactory task.ControllerFactory) (cluster StorageCluster, err error) {
		return storage1, nil
	}
	mgr1.mutex.Unlock()

	storage1.EXPECT().Start().Return(nil)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.StorageConfigChanged,
		Key:   "/storage/test",
		Value: encoding.JSONMarshal(&config.StorageCluster{Name: "test"}),
	})
	time.Sleep(100 * time.Millisecond)

	// case 1: unmarshal cfg err
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: []byte("value"),
	})
	// case 2: database name is empty
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: encoding.JSONMarshal(&models.Database{}),
	})
	data := encoding.JSONMarshal(&models.Database{
		Name:          "test",
		Storage:       "test",
		NumOfShard:    3,
		ReplicaFactor: 2,
		Option:        option.DatabaseOption{},
	})
	// case 3: get shard assign err
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: data,
	})
	// case 4: modify shard assign err
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte("value"), nil)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: data,
	})
	storage1.EXPECT().GetLiveNodes().Return(nil, fmt.Errorf("err"))
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(encoding.JSONMarshal(&models.ShardAssignment{}), nil)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: data,
	})
	// case 5: create shard assign err
	storage1.EXPECT().GetLiveNodes().Return(nil, fmt.Errorf("err"))
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, state.ErrNotExist)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: data,
	})
	// case 6: trigger modify event
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(encoding.JSONMarshal(&models.ShardAssignment{
		Shards: map[models.ShardID]*models.Replica{1: nil, 2: nil, 3: nil},
	}), nil)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.DatabaseConfigChanged,
		Key:   "/database/test",
		Value: data,
	})

	time.Sleep(100 * time.Millisecond)
	storage1.EXPECT().Close()
	mgr.Close()
}

func TestStateManager_ShardAssignment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	repo := state.NewMockRepository(ctrl)
	storage := NewMockStorageCluster(ctrl)
	storage.EXPECT().Close().AnyTimes()
	mgr := NewStateManager(context.TODO(), repo, nil, nil)
	mgr1 := mgr.(*stateManager)
	elector := NewMockReplicaLeaderElector(ctrl)
	mgr1.mutex.Lock()
	mgr1.elector = elector
	mgr1.databases["test"] = models.Database{Storage: "test"}
	mgr1.storages["test"] = storage
	mgr1.mutex.Unlock()
	// case 1: unmarshal err
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.ShardAssignmentChanged,
		Key:   "/shard/assign/test",
		Value: []byte("valuek"),
	})
	// case 2: put state err
	data := encoding.JSONMarshal(&models.ShardAssignment{
		Name:   "test",
		Shards: map[models.ShardID]*models.Replica{1: {Replicas: []models.NodeID{2, 3}}, 2: {Replicas: []models.NodeID{2, 3}}},
	})
	storage.EXPECT().GetState().Return(models.NewStorageState("test"))
	elector.EXPECT().ElectLeader(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.NodeID(2), nil)
	elector.EXPECT().ElectLeader(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.NodeID(0), fmt.Errorf("err"))
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.ShardAssignmentChanged,
		Key:   "/shard/assign/test",
		Value: data,
	})
	// case 2: put state err
	storage.EXPECT().GetState().Return(models.NewStorageState("test"))
	elector.EXPECT().ElectLeader(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.NodeID(2), nil)
	elector.EXPECT().ElectLeader(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.NodeID(0), fmt.Errorf("err"))
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mgr.EmitEvent(&discovery.Event{
		Type:  discovery.ShardAssignmentChanged,
		Key:   "/shard/assign/test",
		Value: data,
	})
	time.Sleep(100 * time.Millisecond)
	mgr.Close()
}

func TestStateManager_createShardAssign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	repo := state.NewMockRepository(ctrl)
	storage := NewMockStorageCluster(ctrl)
	storage.EXPECT().Close().AnyTimes()
	mgr := NewStateManager(context.TODO(), repo, nil, nil)
	mgr1 := mgr.(*stateManager)
	// case 1: get live nodes err
	storage.EXPECT().GetLiveNodes().Return(nil, fmt.Errorf("err"))
	shardAssign, err := mgr1.createShardAssignment(storage, &models.Database{Name: "test"}, -1, -1)
	assert.Error(t, err)
	assert.Nil(t, shardAssign)
	// case 2: no live nodes
	storage.EXPECT().GetLiveNodes().Return(nil, nil)
	shardAssign, err = mgr1.createShardAssignment(storage, &models.Database{Name: "test"}, -1, -1)
	assert.Error(t, err)
	assert.Nil(t, shardAssign)
	// case 3: assign shard err
	storage.EXPECT().GetLiveNodes().Return([]models.StatefulNode{{ID: 1}, {ID: 2}, {ID: 3}}, nil).AnyTimes()
	shardAssign, err = mgr1.createShardAssignment(storage, &models.Database{Name: "test"}, -1, -1)
	assert.Error(t, err)
	assert.Nil(t, shardAssign)
	// case 4: save shard assign err
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	shardAssign, err = mgr1.createShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3, ReplicaFactor: 2},
		-1, -1)
	assert.Error(t, err)
	assert.Nil(t, shardAssign)
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	storage.EXPECT().SaveDatabaseAssignment(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	shardAssign, err = mgr1.createShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3, ReplicaFactor: 2},
		-1, -1)
	assert.Error(t, err)
	assert.Nil(t, shardAssign)
	// case 5:ok
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	storage.EXPECT().SaveDatabaseAssignment(gomock.Any(), gomock.Any()).Return(nil)
	shardAssign, err = mgr1.createShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3, ReplicaFactor: 2},
		-1, -1)
	assert.NoError(t, err)
	assert.NotNil(t, shardAssign)
}

func TestStateManager_modifyShardAssign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	repo := state.NewMockRepository(ctrl)
	storage := NewMockStorageCluster(ctrl)
	storage.EXPECT().Close().AnyTimes()
	mgr := NewStateManager(context.TODO(), repo, nil, nil)
	mgr1 := mgr.(*stateManager)
	// case 1: no impl
	assert.Panics(t, func() {
		_ = mgr1.modifyShardAssignment(storage,
			&models.Database{Name: "test"},
			&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	})
	// case 2: get live nodes err
	storage.EXPECT().GetLiveNodes().Return(nil, fmt.Errorf("err"))
	err := mgr1.modifyShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3},
		&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	assert.Error(t, err)
	// case 3: no live nodes
	storage.EXPECT().GetLiveNodes().Return(nil, nil)
	err = mgr1.modifyShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3},
		&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	assert.Error(t, err)
	// case 4: modify err
	storage.EXPECT().GetLiveNodes().Return([]models.StatefulNode{{ID: 1}, {ID: 2}, {ID: 3}}, nil).AnyTimes()
	err = mgr1.modifyShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3},
		&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	assert.Error(t, err)

	// case 5: save err
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	err = mgr1.modifyShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3, ReplicaFactor: 2},
		&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	assert.Error(t, err)
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	storage.EXPECT().SaveDatabaseAssignment(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	err = mgr1.modifyShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3, ReplicaFactor: 2},
		&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	assert.Error(t, err)
	// case 6: ok
	storage.EXPECT().SaveDatabaseAssignment(gomock.Any(), gomock.Any()).Return(nil)
	err = mgr1.modifyShardAssignment(storage,
		&models.Database{Name: "test", NumOfShard: 3, ReplicaFactor: 2},
		&models.ShardAssignment{Shards: map[models.ShardID]*models.Replica{1: {}, 2: {}}})
	assert.NoError(t, err)
}

func TestStateManager_StorageNodeStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	repo := state.NewMockRepository(ctrl)
	storage := NewMockStorageCluster(ctrl)
	storage.EXPECT().Close().AnyTimes()
	mgr := NewStateManager(context.TODO(), repo, nil, nil)
	mgr1 := mgr.(*stateManager)
	mgr1.mutex.Lock()
	mgr1.storages["test"] = storage
	mgr1.mutex.Unlock()
	// case 1: unmarshal err
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeStartup,
		Key:        "/test/1",
		Value:      []byte("dd"),
		Attributes: map[string]string{storageNameKey: "test"},
	})
	// case 2: sync err
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	storage.EXPECT().GetState().Return(models.NewStorageState("test"))
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeStartup,
		Key:        "/test/1",
		Value:      []byte(`{"id":1}`),
		Attributes: map[string]string{storageNameKey: "test"},
	})
	// case 3: change shard state,but shard state not found
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	storage.EXPECT().GetState().Return(&models.StorageState{
		Name:      "test",
		LiveNodes: map[models.NodeID]models.StatefulNode{},
		ShardAssignments: map[string]*models.ShardAssignment{"test": {
			Shards: map[models.ShardID]*models.Replica{1: {Replicas: []models.NodeID{1, 2, 3, 4}}},
		}},
	})
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeStartup,
		Key:        "/test/1",
		Value:      []byte(`{"id":1}`),
		Attributes: map[string]string{storageNameKey: "test"},
	})
	// case 4: change shard state ok
	storage.EXPECT().GetState().Return(&models.StorageState{
		Name:        "test",
		LiveNodes:   map[models.NodeID]models.StatefulNode{},
		ShardStates: map[string]map[models.ShardID]models.ShardState{"test": {1: {}}},
		ShardAssignments: map[string]*models.ShardAssignment{"test": {
			Shards: map[models.ShardID]*models.Replica{1: {Replicas: []models.NodeID{1, 2, 3, 4}}},
		}},
	})
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeStartup,
		Key:        "/test/1",
		Value:      []byte(`{"id":1}`),
		Attributes: map[string]string{storageNameKey: "test"},
	})
	time.Sleep(100 * time.Millisecond)
	mgr.Close()
}

func TestStateManager_StorageNodeFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	repo := state.NewMockRepository(ctrl)
	storage := NewMockStorageCluster(ctrl)
	storage.EXPECT().Close().AnyTimes()
	mgr := NewStateManager(context.TODO(), repo, nil, nil)
	mgr1 := mgr.(*stateManager)
	mgr1.mutex.Lock()
	mgr1.storages["test"] = storage
	mgr1.mutex.Unlock()
	// case 1: unmarshal node id err
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeFailure,
		Key:        "/test/test_1",
		Attributes: map[string]string{storageNameKey: "test"},
	})
	// case 2: sync err
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	storage.EXPECT().GetState().Return(models.NewStorageState("test"))
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeFailure,
		Key:        "/test/1",
		Attributes: map[string]string{storageNameKey: "test"},
	})
	// case 3: change shard state,but elect leader err
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	storage.EXPECT().GetState().Return(&models.StorageState{
		Name:        "test",
		LiveNodes:   map[models.NodeID]models.StatefulNode{},
		ShardStates: map[string]map[models.ShardID]models.ShardState{"test": {1: {Leader: 1}}},
		ShardAssignments: map[string]*models.ShardAssignment{"test": {
			Shards: map[models.ShardID]*models.Replica{1: {Replicas: []models.NodeID{1, 2, 3, 4}}},
		}},
	})
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeFailure,
		Key:        "/test/1",
		Attributes: map[string]string{storageNameKey: "test"},
	})
	// case 4: change shard state ok
	repo.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	storage.EXPECT().GetState().Return(&models.StorageState{
		Name:        "test",
		LiveNodes:   map[models.NodeID]models.StatefulNode{1: {ID: 1}, 2: {ID: 2}},
		ShardStates: map[string]map[models.ShardID]models.ShardState{"test": {1: {Leader: 1}}},
		ShardAssignments: map[string]*models.ShardAssignment{"test": {
			Shards: map[models.ShardID]*models.Replica{1: {Replicas: []models.NodeID{1, 2, 3, 4}}},
		}},
	})
	mgr.EmitEvent(&discovery.Event{
		Type:       discovery.NodeFailure,
		Key:        "/test/1",
		Attributes: map[string]string{storageNameKey: "test"},
	})

	time.Sleep(300 * time.Millisecond)
	mgr.Close()
}
