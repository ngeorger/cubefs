// Copyright 2018 The CubeFS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package master

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/cubefs/cubefs/proto"
	"github.com/cubefs/cubefs/util/errors"
	"github.com/cubefs/cubefs/util/log"
	//"github.com/cubefs/cubefs/blobstore/util/errors"
)

type MasterQuotaManager struct {
	MpQuotaInfoMap map[uint64][]*proto.QuotaReportInfo
	IdQuotaInfoMap map[uint32]*proto.QuotaInfo
	vol            *Vol
	c              *Cluster
	sync.RWMutex
}

func (mqMgr *MasterQuotaManager) createQuota(req *proto.SetMasterQuotaReuqest) (quotaId uint32, err error) {
	mqMgr.Lock()
	defer mqMgr.Unlock()

	if len(mqMgr.IdQuotaInfoMap) >= gConfig.MaxQuotaNumPerVol {
		err = errors.NewErrorf("the number of quota has reached the upper limit %v", len(mqMgr.IdQuotaInfoMap))
		return
	}
	for _, quotaInfo := range mqMgr.IdQuotaInfoMap {
		for _, pathInfo := range req.PathInfos {
			for _, quotaPathInfo := range quotaInfo.PathInfos {
				if pathInfo.RootInode == quotaPathInfo.RootInode {
					err = errors.NewErrorf("path [%v] is the same as quotaId [%v]",
						pathInfo.FullPath, quotaInfo.QuotaId)
					return
				}
				if pathInfo.FullPath == quotaPathInfo.FullPath {
					err = errors.NewErrorf("path [%v] is the same as quotaId [%v]",
						pathInfo.FullPath, quotaInfo.QuotaId)
					return
				}
			}
		}
	}

	if quotaId, err = mqMgr.c.idAlloc.allocateQuotaID(); err != nil {
		return
	}

	var quotaInfo = &proto.QuotaInfo{
		VolName:   req.VolName,
		QuotaId:   quotaId,
		CTime:     time.Now().Unix(),
		PathInfos: make([]proto.QuotaPathInfo, 0, 0),
		MaxFiles:  req.MaxFiles,
		MaxBytes:  req.MaxBytes,
	}

	for _, pathInfo := range req.PathInfos {
		quotaInfo.PathInfos = append(quotaInfo.PathInfos, pathInfo)
	}

	var value []byte
	if value, err = json.Marshal(quotaInfo); err != nil {
		log.LogErrorf("create quota [%v] marsha1 fail [%v].", quotaInfo, err)
		return
	}

	metadata := new(RaftCmd)
	metadata.Op = opSyncSetQuota
	metadata.K = quotaPrefix + strconv.FormatUint(mqMgr.vol.ID, 10) + keySeparator + strconv.FormatUint(uint64(quotaId), 10)
	metadata.V = value

	if err = mqMgr.c.submit(metadata); err != nil {
		log.LogErrorf("create quota [%v] submit fail [%v].", quotaInfo, err)
		return
	}

	// for _, pathInfo := range req.PathInfos {
	// 	var inodes = make([]uint64, 0)
	// 	inodes = append(inodes, pathInfo.RootInode)
	// 	request := &proto.BatchSetMetaserverQuotaReuqest{
	// 		PartitionId: pathInfo.PartitionId,
	// 		Inodes:      inodes,
	// 		QuotaId:     quotaId,
	// 	}

	// 	if err = mqMgr.setQuotaToMetaNode(request); err != nil {
	// 		log.LogErrorf("create quota [%v] to metanode fail [%v].", quotaInfo, err)
	// 		return
	// 	}
	// }
	mqMgr.IdQuotaInfoMap[quotaId] = quotaInfo

	log.LogInfof("create quota [%v] success.", quotaInfo)
	return
}

func (mqMgr *MasterQuotaManager) updateQuota(req *proto.UpdateMasterQuotaReuqest) (err error) {
	mqMgr.Lock()
	defer mqMgr.Unlock()
	quotaInfo, isFind := mqMgr.IdQuotaInfoMap[req.QuotaId]
	if !isFind {
		log.LogErrorf("vol [%v] quota quotaId [%v] is not exist.", mqMgr.vol.Name, req.QuotaId)
		err = errors.New("quota is not exist.")
		return
	}

	quotaInfo.MaxFiles = req.MaxFiles
	quotaInfo.MaxBytes = req.MaxBytes

	var value []byte
	if value, err = json.Marshal(quotaInfo); err != nil {
		log.LogErrorf("update quota [%v] marsha1 fail [%v].", quotaInfo, err)
		return
	}

	metadata := new(RaftCmd)
	metadata.Op = opSyncSetQuota
	metadata.K = quotaPrefix + strconv.FormatUint(mqMgr.vol.ID, 10) + keySeparator + strconv.FormatUint(uint64(quotaInfo.QuotaId), 10)
	metadata.V = value

	if err = mqMgr.c.submit(metadata); err != nil {
		log.LogErrorf("update quota [%v] submit fail [%v].", quotaInfo, err)
		return
	}

	log.LogInfof("update quota [%v] success.", *quotaInfo)
	return
}

func (mqMgr *MasterQuotaManager) listQuota() (resp *proto.ListMasterQuotaResponse) {
	mqMgr.RLock()
	defer mqMgr.RUnlock()
	resp = &proto.ListMasterQuotaResponse{}
	resp.Quotas = make([]*proto.QuotaInfo, 0)
	for _, info := range mqMgr.IdQuotaInfoMap {
		resp.Quotas = append(resp.Quotas, info)
	}
	return
}

func (mqMgr *MasterQuotaManager) getQuota(quotaId uint32) (quotaInfo *proto.QuotaInfo, err error) {
	mqMgr.RLock()
	defer mqMgr.RUnlock()
	quotaInfo, isFind := mqMgr.IdQuotaInfoMap[quotaId]
	if !isFind {
		err = errors.New("quota is not exist.")
		return nil, err
	}

	return quotaInfo, nil
}

func (mqMgr *MasterQuotaManager) deleteQuota(quotaId uint32) (err error) {
	mqMgr.Lock()
	defer mqMgr.Unlock()

	quotaInfo, isFind := mqMgr.IdQuotaInfoMap[quotaId]
	if !isFind {
		log.LogErrorf("vol [%v] quota quotaId [%v] is not exist.", mqMgr.vol.Name, quotaId)
		err = errors.New("quota is not exist.")
		return
	}

	var value []byte
	if value, err = json.Marshal(quotaInfo); err != nil {
		log.LogErrorf("delete quota [%v] marsha1 fail [%v].", quotaInfo, err)
		return
	}
	metadata := new(RaftCmd)
	metadata.Op = opSyncDeleteQuota
	metadata.K = quotaPrefix + strconv.FormatUint(mqMgr.vol.ID, 10) + keySeparator + strconv.FormatUint(uint64(quotaInfo.QuotaId), 10)
	metadata.V = value

	if err = mqMgr.c.submit(metadata); err != nil {
		log.LogErrorf("delete quota [%v] submit fail [%v].", quotaInfo, err)
		return
	}

	delete(mqMgr.IdQuotaInfoMap, quotaInfo.QuotaId)
	log.LogInfof("deleteQuota: idmap len [%v]", len(mqMgr.IdQuotaInfoMap))
	return
}

func (mqMgr *MasterQuotaManager) getQuotaInfoById(quotaId uint32) (quotaInfo *proto.QuotaInfo, err error) {
	var isFind bool
	mqMgr.RLock()
	defer mqMgr.RUnlock()
	if quotaInfo, isFind = mqMgr.IdQuotaInfoMap[quotaId]; isFind {
		err = nil
		return
	} else {
		err = errors.New("quota is exist.")
	}
	return
}

func (mqMgr *MasterQuotaManager) DeleteQuotaInfoById(quotaId uint32) {
	mqMgr.Lock()
	defer mqMgr.Unlock()

	_, isFind := mqMgr.IdQuotaInfoMap[quotaId]
	if isFind {
		delete(mqMgr.IdQuotaInfoMap, quotaId)
		log.LogInfof("DeleteQuotaInfoById delete quotaId [%v] success.", quotaId)
	}
	return
}

func (mqMgr *MasterQuotaManager) quotaUpdate(report *proto.MetaPartitionReport) {
	var (
		quotaInfo = &proto.QuotaInfo{}
		id        uint32
	)

	mqMgr.Lock()
	defer mqMgr.Unlock()

	mpId := report.PartitionID

	if !report.IsLeader {
		return
	}

	mqMgr.MpQuotaInfoMap[mpId] = report.QuotaReportInfos

	for _, quotaInfo = range mqMgr.IdQuotaInfoMap {
		quotaInfo.UsedInfo.UsedFiles = 0
		quotaInfo.UsedInfo.UsedBytes = 0
	}
	deleteQuotaIds := make([]uint32, 0, 0)
	for mpId, reportInfos := range mqMgr.MpQuotaInfoMap {
		for _, info := range reportInfos {
			if _, isFind := mqMgr.IdQuotaInfoMap[info.QuotaId]; !isFind {
				deleteQuotaIds = append(deleteQuotaIds, info.QuotaId)
				continue
			}
			log.LogDebugf("[quotaUpdate] mpId [%v] quotaId [%v] reportinfo [%v]", mpId, info.QuotaId, info.UsedInfo)
			quotaInfo = mqMgr.IdQuotaInfoMap[info.QuotaId]
			quotaInfo.UsedInfo.Add(&info.UsedInfo)
		}
	}
	if len(deleteQuotaIds) != 0 {
		log.LogWarnf("[quotaUpdate] quotaIds [%v] is delete", deleteQuotaIds)
	}
	for id, quotaInfo = range mqMgr.IdQuotaInfoMap {
		if quotaInfo.IsOverQuotaFiles() {
			quotaInfo.LimitedInfo.LimitedFiles = true
		} else {
			quotaInfo.LimitedInfo.LimitedFiles = false
		}
		if quotaInfo.IsOverQuotaBytes() {
			quotaInfo.LimitedInfo.LimitedBytes = true
		} else {
			quotaInfo.LimitedInfo.LimitedBytes = false
		}
		log.LogDebugf("[quotaUpdate] quotaId [%v] quotaInfo [%v]", id, quotaInfo)
	}
	return
}

func (mqMgr *MasterQuotaManager) getQuotaHbInfos() (infos []*proto.QuotaHeartBeatInfo) {
	mqMgr.RLock()
	defer mqMgr.RUnlock()
	for quotaId, quotaInfo := range mqMgr.IdQuotaInfoMap {
		var info = &proto.QuotaHeartBeatInfo{}
		info.VolName = mqMgr.vol.Name
		info.QuotaId = quotaId
		info.LimitedInfo.LimitedFiles = quotaInfo.LimitedInfo.LimitedFiles
		info.LimitedInfo.LimitedBytes = quotaInfo.LimitedInfo.LimitedBytes
		infos = append(infos, info)
		log.LogDebugf("getQuotaHbInfos info %v", info)
	}

	return
}

func (mqMgr *MasterQuotaManager) HasQuota() bool {
	mqMgr.RLock()
	defer mqMgr.RUnlock()

	if len(mqMgr.IdQuotaInfoMap) == 0 {
		return false
	}
	return true
}
