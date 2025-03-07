package gdconf

import (
	"fmt"
	"os"

	"github.com/gucooing/hkrpg-go/pkg/logger"
	"github.com/hjson/hjson-go/v4"
)

type MultiplePathAvatarConfig struct {
	BaseAvatarID     uint32              `json:"BaseAvatarID"`     // 角色id
	AvatarID         uint32              `json:"AvatarID"`         // 命途id
	UnlockConditions []*UnlockConditions `json:"UnlockConditions"` // 解锁条件
}

func (g *GameDataConfig) loadMultiplePathAvatarConfig() {
	g.MultiplePathAvatarConfigMap = make(map[uint32]*MultiplePathAvatarConfig)
	playerElementsFilePath := g.dataPrefix + "MultiplePathAvatarConfig.json"
	playerElementsFile, err := os.ReadFile(playerElementsFilePath)
	if err != nil {
		info := fmt.Sprintf("open file error: %v", err)
		panic(info)
	}

	err = hjson.Unmarshal(playerElementsFile, &g.MultiplePathAvatarConfigMap)
	if err != nil {
		info := fmt.Sprintf("parse file error: %v", err)
		panic(info)
	}
	logger.Info("load %v MultiplePathAvatarConfig", len(g.MultiplePathAvatarConfigMap))
}

func GetMultiplePathAvatarConfigMap() map[uint32]*MultiplePathAvatarConfig {
	return CONF.MultiplePathAvatarConfigMap
}

func GetMultiplePathAvatarConfig(avatarID uint32) *MultiplePathAvatarConfig {
	return CONF.MultiplePathAvatarConfigMap[avatarID]
}
