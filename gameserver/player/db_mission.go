package player

import (
	"github.com/gucooing/hkrpg-go/pkg/gdconf"
	"github.com/gucooing/hkrpg-go/protocol/proto"
	spb "github.com/gucooing/hkrpg-go/protocol/server"
)

func newMission() *spb.Mission {
	return &spb.Mission{}
}

func (g *GamePlayer) GetMission() *spb.Mission {
	db := g.GetBasicBin()
	if db.Mission == nil {
		db.Mission = newMission()
	}
	return db.Mission
}

func (g *GamePlayer) GetMainMission() *spb.MainMission {
	db := g.GetMission()
	if db.MainMission == nil {
		db.MainMission = &spb.MainMission{}
	}
	return db.MainMission
}

func (g *GamePlayer) GetMainMissionList() map[uint32]*spb.MissionInfo {
	db := g.GetMainMission()
	if db.MainMissionList == nil {
		db.MainMissionList = make(map[uint32]*spb.MissionInfo)
	}
	return db.MainMissionList
}

func (g *GamePlayer) GetSubMainMissionList() map[uint32]*spb.MissionInfo {
	db := g.GetMainMission()
	if db.SubMissionList == nil {
		db.SubMissionList = make(map[uint32]*spb.MissionInfo)
	}
	return db.SubMissionList
}

func (g *GamePlayer) GetSubMainMissionById(id uint32) *spb.MissionInfo {
	db := g.GetSubMainMissionList()
	return db[id]
}

func (g *GamePlayer) GetFinishMainMissionList() map[uint32]*spb.MissionInfo {
	db := g.GetMainMission()
	if db.FinishMainMissionList == nil {
		db.FinishMainMissionList = make(map[uint32]*spb.MissionInfo)
	}
	return db.FinishMainMissionList
}

func (g *GamePlayer) GetFinishSubMainMissionList() map[uint32]*spb.MissionInfo {
	db := g.GetMainMission()
	if db.FinishSubMissionList == nil {
		db.FinishSubMissionList = make(map[uint32]*spb.MissionInfo)
	}
	return db.FinishSubMissionList
}

func (g *GamePlayer) GetFinishSubMainMissionById(id uint32) *spb.MissionInfo {
	db := g.GetFinishSubMainMissionList()
	return db[id]
}

// 将子任务转成完成状态
func (g *GamePlayer) UpSubMainMission(subMissionId uint32) bool {
	subMainMissionList := g.GetSubMainMissionList()
	subMission := subMainMissionList[subMissionId]
	finishSubMainMissionList := g.GetFinishSubMainMissionList()
	if subMission == nil {
		return false
	}

	finishSubMainMissionList[subMissionId] = &spb.MissionInfo{
		MissionId: subMission.MissionId,
		Progress:  subMission.Progress + 1,
		Status:    spb.MissionStatus_MISSION_FINISH,
	}
	delete(subMainMissionList, subMissionId)
	return true
}

// 处理战斗任务
func (g *GamePlayer) UpBattleSubMission(req *proto.PVEBattleResultCsReq) {
	db := g.GetBattleBackupById(req.BattleId)
	if db.EventId == 0 {
		return
	}
	for id := range g.GetSubMainMissionList() {
		conf := gdconf.GetSubMainMissionById(id)
		if conf == nil {
			continue
		}
		switch conf.FinishType {
		case "StageWin":
			if req.EndStatus == proto.BattleEndStatus_BATTLE_END_WIN && db.EventId == conf.ParamInt1 {
				g.FinishSubMission(id)
			}
		}
	}
}

// 完成子任务并拉取下一个任务和通知
func (g *GamePlayer) FinishSubMission(missionId uint32) {
	// 先完成子任务
	if !g.UpSubMainMission(missionId) {
		return
	}
	nextList := make([]uint32, 0)
	finishSubMainMissionList := g.GetFinishSubMainMissionList()
	subMainMissionList := g.GetSubMainMissionList()
	subMissionConf := gdconf.GetSubMainMissionById(missionId)
	if subMissionConf == nil {
		return
	}
	conf := gdconf.GetGoppMainMissionById(subMissionConf.MainMissionID)
	if conf == nil {
		return
	}
	for _, confSubMission := range conf.SubMissionList {
		var isNext = false
		if subMainMissionList[confSubMission.ID] != nil || finishSubMainMissionList[confSubMission.ID] != nil {
			continue
		}
		for _, takeParamId := range confSubMission.TakeParamIntList {
			if finishSubMainMissionList[takeParamId] != nil {
				isNext = true
			} else {
				isNext = false
				break
			}
		}
		if isNext {
			nextList = append(nextList, confSubMission.ID)
			subMainMissionList[confSubMission.ID] = &spb.MissionInfo{
				MissionId: confSubMission.ID,
				Progress:  0,
				Status:    spb.MissionStatus_MISSION_DOING,
			}
		}
	}
	// 通知状态
	g.MissionPlayerSyncScNotify(nextList, []uint32{missionId}) // 发送通知
}

// 登录事件-自动接取任务
func (g *GamePlayer) ReadyMission() {
	g.ReadyMainMission() // 主线检查
}

// 主线检查
func (g *GamePlayer) ReadyMainMission() {
	mainMissionList := g.GetMainMissionList()
	finishMainMissionList := g.GetFinishMainMissionList()
	subMainMissionList := g.GetSubMainMissionList()
	finishSubMainMissionList := g.GetFinishSubMainMissionList()
	conf := gdconf.GetMainMission()
	for id, mission := range conf {
		if g.IsReceiveMission(mission, mainMissionList, finishMainMissionList) {
			goppConf := gdconf.GetGoppMainMissionById(id)
			if goppConf == nil {
				continue
			}
			mainMissionList[id] = &spb.MissionInfo{
				MissionId: id,
				Progress:  0,
				Status:    spb.MissionStatus_MISSION_DOING,
			}
			for _, subId := range goppConf.StartSubMissionList {
				if finishSubMainMissionList[subId] != nil {
					continue
				}
				subMainMissionList[subId] = &spb.MissionInfo{
					MissionId: subId,
					Progress:  0,
					Status:    spb.MissionStatus_MISSION_DOING,
				}
			}
		}
	}
}

func (g *GamePlayer) IsReceiveMission(mission *gdconf.MainMission, mainMissionList, finishMainMissionList map[uint32]*spb.MissionInfo) bool {
	var isReceive = false
	if mission == nil || mainMissionList == nil || finishMainMissionList == nil || mission.TakeParam == nil {
		return false
	}
	if mainMissionList[mission.MainMissionID] != nil || finishMainMissionList[mission.MainMissionID] != nil { // 过滤已接取已完成的
		return false
	}
	for _, take := range mission.TakeParam {
		switch take.Type {
		case "Auto":
			isReceive = true
		default:
			isReceive = false
		}
	}

	return isReceive
}
