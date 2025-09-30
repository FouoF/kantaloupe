package informermanager

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestMultiClusterInformerManager(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	transforms := map[schema.GroupVersionResource]cache.TransformFunc{
		NodeGVR: NodeTransformFunc,
		PodGVR:  PodTransformFunc,
	}

	manager := NewMultiClusterInformerManager(stopCh, transforms)

	t.Run("ForCluster", func(_ *testing.T) {
		cluster := "test-cluster"
		client := fake.NewSimpleClientset()
		resync := 10 * time.Second

		singleManager := manager.ForCluster(cluster, client, resync)
		if singleManager == nil {
			t.Fatalf("ForCluster() returned nil")
		}

		if !manager.IsManagerExist(cluster) {
			t.Fatalf("IsManagerExist() returned false for existing cluster")
		}
	})

	t.Run("GetSingleClusterManager", func(t *testing.T) {
		cluster := "test-cluster"
		singleManager := manager.GetSingleClusterManager(cluster)
		if singleManager == nil {
			t.Fatalf("GetSingleClusterManager() returned nil for existing cluster")
		}

		nonExistentCluster := "non-existent-cluster"
		singleManager = manager.GetSingleClusterManager(nonExistentCluster)
		if singleManager != nil {
			t.Fatalf("GetSingleClusterManager() returned non-nil for non-existent cluster")
		}
	})

	t.Run("Start and Stop", func(t *testing.T) {
		cluster := "test-cluster-2"
		client := fake.NewSimpleClientset()
		resync := 10 * time.Second

		manager.ForCluster(cluster, client, resync)
		manager.Start(cluster)

		manager.Stop(cluster)

		if manager.IsManagerExist(cluster) {
			t.Fatalf("IsManagerExist() returned true after Stop()")
		}
	})

	t.Run("WaitForCacheSync", func(t *testing.T) {
		cluster := "test-cluster-3"
		client := fake.NewSimpleClientset()
		resync := 10 * time.Millisecond
		singleManager := manager.ForCluster(cluster, client, resync)
		manager.Start(cluster)

		_, _ = singleManager.Lister(PodGVR)
		_, _ = singleManager.Lister(NodeGVR)

		time.Sleep(100 * time.Millisecond)

		result := manager.WaitForCacheSync(cluster)
		if result == nil {
			t.Fatalf("WaitForCacheSync() returned nil result")
		}

		for gvr, synced := range result {
			t.Logf("Resource %v synced: %v", gvr, synced)
		}

		manager.Stop(cluster)
	})

	t.Run("WaitForCacheSyncWithTimeout", func(t *testing.T) {
		cluster := "test-cluster-4"
		client := fake.NewSimpleClientset()
		resync := 10 * time.Millisecond
		singleManager := manager.ForCluster(cluster, client, resync)
		manager.Start(cluster)

		_, _ = singleManager.Lister(PodGVR)
		_, _ = singleManager.Lister(NodeGVR)

		timeout := 100 * time.Millisecond
		result := manager.WaitForCacheSyncWithTimeout(cluster, timeout)
		if result == nil {
			t.Fatalf("WaitForCacheSyncWithTimeout() returned nil result")
		}

		for gvr, synced := range result {
			t.Logf("Resource %v synced: %v", gvr, synced)
		}

		manager.Stop(cluster)
	})

	t.Run("WaitForCacheSync and WaitForCacheSyncWithTimeout with non-existent cluster", func(t *testing.T) {
		nonExistentCluster := "non-existent-cluster"

		result1 := manager.WaitForCacheSync(nonExistentCluster)
		if result1 != nil {
			t.Fatalf("WaitForCacheSync() returned non-nil for non-existent cluster")
		}

		result2 := manager.WaitForCacheSyncWithTimeout(nonExistentCluster, 1*time.Second)
		if result2 != nil {
			t.Fatalf("WaitForCacheSyncWithTimeout() returned non-nil for non-existent cluster")
		}
	})
}

func TestGetInstance(t *testing.T) {
	instance1 := GetInstance()
	instance2 := GetInstance()

	if instance1 != instance2 {
		t.Fatalf("GetInstance() returned different instances")
	}
}

func TestStopInstance(_ *testing.T) {
	StopInstance()
	// Ensure StopInstance doesn't panic
	StopInstance()
}
