package master

import (
	"testing"
)

func TestGetDataPartitions(t *testing.T) {
	testVolName := "ltptest"
	_, err := testMc.ClientAPI().GetDataPartitions(testVolName)
	if err != nil {
		t.Fatalf("GetDataPartitions failed, err %v", err)
	}
}

func TestGetMetaPartition(t *testing.T) {
	// get meta node info
	cv, err := testMc.AdminAPI().GetCluster()
	if err != nil {
		t.Fatalf("Get cluster failed: err(%v), cluster(%v)", err, cv)
	}
	if len(cv.MetaNodes) < 1 {
		t.Fatalf("metanodes[] len < 1")
	}
	maxMetaPartitionId := cv.MaxMetaPartitionID
	testMetaPartitionID := maxMetaPartitionId
	_, err = testMc.ClientAPI().GetMetaPartition(testMetaPartitionID)
	if err != nil {
		t.Fatalf("GetMetaPartition failed, err %v", err)
	}
}

func TestGetMetaPartitions(t *testing.T) {
	testVolName := "ltptest"
	_, err := testMc.ClientAPI().GetMetaPartitions(testVolName)
	if err != nil {
		t.Fatalf("GetMetaPartitions failed, err %v", err)
	}
}

func TestApplyVolMutex(t *testing.T) {
	testVolName := "ltptest"
	err := testMc.ClientAPI().ApplyVolMutex(testVolName)
	if err == nil {
		t.Fatalf("expected err, but nil")
	}
	if err.Error() != "vol write mutex is unable" {
		t.Fatalf("expected err: 'vol write mutex is unable', but it's not")
	}
}

func TestReleaseVolMutex(t *testing.T) {
	testVolName := "ltptest"
	err := testMc.ClientAPI().ReleaseVolMutex(testVolName)
	if err == nil {
		t.Fatalf("expected err, but nil")
	}
	if err.Error() != "vol write mutex is unable" {
		t.Fatalf("expected err: 'vol write mutex is unable', but it's not")
	}
}