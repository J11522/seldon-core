package scheduler

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/seldonio/seldon-core/scheduler/apis/mlops/agent"
	pb "github.com/seldonio/seldon-core/scheduler/apis/mlops/scheduler"
	"github.com/seldonio/seldon-core/scheduler/pkg/scheduler/filters"
	"github.com/seldonio/seldon-core/scheduler/pkg/scheduler/sorters"
	"github.com/seldonio/seldon-core/scheduler/pkg/store"
	log "github.com/sirupsen/logrus"
)

type mockStore struct {
	models            map[string]*store.ModelSnapshot
	servers           []*store.ServerSnapshot
	scheduledServer   string
	scheduledReplicas []int
}

func (f mockStore) RemoveModel(modelKey string) error {
	panic("implement me")
}

func (f mockStore) UpdateModel(config *pb.ModelDetails) error {
	panic("implement me")
}

func (f mockStore) GetModel(key string) (*store.ModelSnapshot, error) {
	return f.models[key], nil
}

func (f mockStore) GetServers() ([]*store.ServerSnapshot, error) {
	return f.servers, nil
}

func (f mockStore) GetServer(serverKey string) (*store.ServerSnapshot, error) {
	panic("implement me")
}

func (f *mockStore) UpdateLoadedModels(modelKey string, version string, serverKey string, replicas []*store.ServerReplica) error {
	f.scheduledServer = serverKey
	var replicaIdxs []int
	for _, rep := range replicas {
		replicaIdxs = append(replicaIdxs, rep.GetReplicaIdx())
	}
	f.scheduledReplicas = replicaIdxs
	return nil
}

func (f mockStore) UpdateModelState(modelKey string, version string, serverKey string, replicaIdx int, availableMemory *uint64, state store.ModelReplicaState) error {
	panic("implement me")
}

func (f mockStore) AddServerReplica(request *agent.AgentSubscribeRequest) error {
	panic("implement me")
}

func (f mockStore) RemoveServerReplica(serverName string, replicaIdx int) ([]string, error) {
	panic("implement me")
}

func TestScheduler(t *testing.T) {
	logger := log.New()
	g := NewGomegaWithT(t)

	newTestModel := func(name string, requiredMemory uint64, requirements []string, server *string, replicas uint32, loadedModels []int, deleted bool, scheduledServer string) *store.ModelSnapshot {
		config := &pb.ModelDetails{
			Name:         name,
			MemoryBytes:  &requiredMemory,
			Requirements: requirements,
			Server:       server,
			Replicas:     replicas,
		}
		rmap := make(map[int]store.ModelReplicaState)
		for _, ridx := range loadedModels {
			rmap[ridx] = store.Loaded
		}
		return &store.ModelSnapshot{
			Name:     name,
			Versions: []*store.ModelVersion{store.NewModelVersion(config, scheduledServer, rmap, false, store.ModelProgressing)},
			Deleted:  deleted,
		}
	}

	gsr := func(replicaIdx int, availableMemory uint64, capabilities []string) *store.ServerReplica {
		return store.NewServerReplica("svc", 8080, 5001, replicaIdx, nil, capabilities, availableMemory, availableMemory, nil, true)
	}

	type test struct {
		name              string
		model             *store.ModelSnapshot
		servers           []*store.ServerSnapshot
		serverFilters     []ServerFilter
		replicaFilters    []ReplicaFilter
		serverSorts       []sorters.ServerSorter
		replicaSort       []sorters.ReplicaSorter
		scheduled         bool
		scheduledServer   string
		scheduledReplicas []int
	}

	tests := []test{
		{
			name:  "SmokeTest",
			model: newTestModel("model1", 100, []string{"sklearn"}, nil, 1, []int{}, false, ""),
			servers: []*store.ServerSnapshot{
				{
					Name:     "server1",
					Replicas: map[int]*store.ServerReplica{0: gsr(0, 200, []string{"sklearn"})},
					Shared:   true,
				},
			},
			serverFilters:     []ServerFilter{filters.SharingServerFilter{}},
			replicaFilters:    []ReplicaFilter{filters.RequirementsReplicaFilter{}, filters.AvailableMemoryFilter{}},
			serverSorts:       []sorters.ServerSorter{},
			replicaSort:       []sorters.ReplicaSorter{sorters.ModelAlreadyLoadedSorter{}},
			scheduled:         true,
			scheduledServer:   "server1",
			scheduledReplicas: []int{0},
		},
		{
			name:  "ReplicasTwo",
			model: newTestModel("model1", 100, []string{"sklearn"}, nil, 2, []int{}, false, ""),
			servers: []*store.ServerSnapshot{
				{
					Name:     "server1",
					Replicas: map[int]*store.ServerReplica{0: gsr(0, 200, []string{"sklearn"})},
					Shared:   true,
				},
				{
					Name: "server2",
					Replicas: map[int]*store.ServerReplica{
						0: gsr(0, 200, []string{"sklearn"}),
						1: gsr(1, 200, []string{"sklearn"}),
					},
					Shared: true,
				},
			},
			serverFilters:     []ServerFilter{filters.SharingServerFilter{}},
			replicaFilters:    []ReplicaFilter{filters.RequirementsReplicaFilter{}, filters.AvailableMemoryFilter{}},
			serverSorts:       []sorters.ServerSorter{},
			replicaSort:       []sorters.ReplicaSorter{sorters.ModelAlreadyLoadedSorter{}},
			scheduled:         true,
			scheduledServer:   "server2",
			scheduledReplicas: []int{0, 1},
		},
		{
			name:  "NotEnoughReplicas",
			model: newTestModel("model1", 100, []string{"sklearn"}, nil, 2, []int{}, false, ""),
			servers: []*store.ServerSnapshot{
				{
					Name:     "server1",
					Replicas: map[int]*store.ServerReplica{0: gsr(0, 200, []string{"sklearn"})},
					Shared:   true,
				},
				{
					Name: "server2",
					Replicas: map[int]*store.ServerReplica{
						0: gsr(0, 200, []string{"sklearn"}),
						1: gsr(1, 200, []string{"foo"}),
					},
					Shared: true,
				},
			},
			serverFilters:  []ServerFilter{filters.SharingServerFilter{}},
			replicaFilters: []ReplicaFilter{filters.RequirementsReplicaFilter{}, filters.AvailableMemoryFilter{}},
			serverSorts:    []sorters.ServerSorter{},
			replicaSort:    []sorters.ReplicaSorter{sorters.ModelAlreadyLoadedSorter{}},
			scheduled:      false,
		},
		{
			name:  "MemoryOneServer",
			model: newTestModel("model1", 100, []string{"sklearn"}, nil, 1, []int{}, false, ""),
			servers: []*store.ServerSnapshot{
				{
					Name:     "server1",
					Replicas: map[int]*store.ServerReplica{0: gsr(0, 50, []string{"sklearn"})},
					Shared:   true,
				},
				{
					Name: "server2",
					Replicas: map[int]*store.ServerReplica{
						0: gsr(0, 200, []string{"sklearn"}),
					},
					Shared: true,
				},
			},
			serverFilters:     []ServerFilter{filters.SharingServerFilter{}},
			replicaFilters:    []ReplicaFilter{filters.RequirementsReplicaFilter{}, filters.AvailableMemoryFilter{}},
			serverSorts:       []sorters.ServerSorter{},
			replicaSort:       []sorters.ReplicaSorter{sorters.ModelAlreadyLoadedSorter{}},
			scheduled:         true,
			scheduledServer:   "server2",
			scheduledReplicas: []int{0},
		},
		{
			name:  "ModelsLoaded",
			model: newTestModel("model1", 100, []string{"sklearn"}, nil, 2, []int{1}, false, ""),
			servers: []*store.ServerSnapshot{
				{
					Name:     "server1",
					Replicas: map[int]*store.ServerReplica{0: gsr(0, 50, []string{"sklearn"})},
					Shared:   true,
				},
				{
					Name: "server2",
					Replicas: map[int]*store.ServerReplica{
						0: gsr(0, 200, []string{"sklearn"}),
						1: gsr(1, 200, []string{"sklearn"}),
					},
					Shared: true,
				},
			},
			serverFilters:     []ServerFilter{filters.SharingServerFilter{}},
			replicaFilters:    []ReplicaFilter{filters.RequirementsReplicaFilter{}, filters.AvailableMemoryFilter{}},
			serverSorts:       []sorters.ServerSorter{},
			replicaSort:       []sorters.ReplicaSorter{sorters.ModelAlreadyLoadedSorter{}},
			scheduled:         true,
			scheduledServer:   "server2",
			scheduledReplicas: []int{1, 0},
		},
		{
			name:  "ModelUnLoaded",
			model: newTestModel("model1", 100, []string{"sklearn"}, nil, 2, []int{1}, true, "server2"),
			servers: []*store.ServerSnapshot{
				{
					Name: "server2",
					Replicas: map[int]*store.ServerReplica{
						0: gsr(0, 200, []string{"sklearn"}),
						1: gsr(1, 200, []string{"sklearn"}),
					},
					Shared: true,
				},
			},
			serverFilters:     []ServerFilter{},
			replicaFilters:    []ReplicaFilter{},
			serverSorts:       []sorters.ServerSorter{},
			replicaSort:       []sorters.ReplicaSorter{},
			scheduled:         true,
			scheduledServer:   "server2",
			scheduledReplicas: nil,
		},
	}

	newMockStore := func(model *store.ModelSnapshot, servers []*store.ServerSnapshot) *mockStore {
		modelMap := make(map[string]*store.ModelSnapshot)
		modelMap[model.Name] = model
		return &mockStore{
			models:  modelMap,
			servers: servers,
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			//t.Logf("Running schedule test %d",tidx)
			mockStore := newMockStore(test.model, test.servers)
			scheduler := NewSimpleScheduler(logger, mockStore, test.serverFilters, test.replicaFilters, test.serverSorts, test.replicaSort)
			err := scheduler.Schedule(test.model.Name)
			if test.scheduled {
				g.Expect(err).To(BeNil())
				g.Expect(test.scheduledServer).To(Equal(mockStore.scheduledServer))
				g.Expect(test.scheduledReplicas).To(Equal(mockStore.scheduledReplicas))
			} else {
				g.Expect(err).ToNot(BeNil())
			}
		})

	}

}
