package model

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common/config"
)

// SpecialQuota 用户的专项余额
type SpecialQuota struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index:idx_user_channel_model,priority:1"`
	ChannelId int    `json:"channel_id" gorm:"index:idx_user_channel_model,priority:2"` // 可以为0表示不限渠道
	Model     string `json:"model" gorm:"index:idx_user_channel_model,priority:3"`       // 模型名称
	Quota     int64  `json:"quota" gorm:"bigint;default:0"`
}

// GetUserSpecialQuota 获取用户针对特定渠道和模型的专项余额
func GetUserSpecialQuota(userId int, channelId int, model string) (int64, error) {
	var specialQuota SpecialQuota
	err := DB.Where("user_id = ? AND channel_id = ? AND model = ?", userId, channelId, model).First(&specialQuota).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return specialQuota.Quota, nil
}

// GetAllUserSpecialQuotas 获取用户所有的专项余额
func GetAllUserSpecialQuotas(userId int) ([]*SpecialQuota, error) {
	var specialQuotas []*SpecialQuota
	err := DB.Where("user_id = ? AND quota > 0", userId).Order("id desc").Find(&specialQuotas).Error
	return specialQuotas, err
}

// IncreaseSpecialQuota 增加用户的专项余额
func IncreaseSpecialQuota(userId int, channelId int, model string, quota int64) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		// 对于专项余额，暂时不支持批量更新，直接更新
		return increaseSpecialQuota(userId, channelId, model, quota)
	}
	return increaseSpecialQuota(userId, channelId, model, quota)
}

func increaseSpecialQuota(userId int, channelId int, model string, quota int64) error {
	var specialQuota SpecialQuota
	err := DB.Where("user_id = ? AND channel_id = ? AND model = ?", userId, channelId, model).First(&specialQuota).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录不存在，创建新记录
			specialQuota = SpecialQuota{
				UserId:    userId,
				ChannelId: channelId,
				Model:     model,
				Quota:     quota,
			}
			return DB.Create(&specialQuota).Error
		}
		return err
	}
	// 记录存在，更新quota
	return DB.Model(&specialQuota).Update("quota", gorm.Expr("quota + ?", quota)).Error
}

// DecreaseSpecialQuota 减少用户的专项余额
func DecreaseSpecialQuota(userId int, channelId int, model string, quota int64) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		return decreaseSpecialQuota(userId, channelId, model, quota)
	}
	return decreaseSpecialQuota(userId, channelId, model, quota)
}

func decreaseSpecialQuota(userId int, channelId int, model string, quota int64) error {
	var specialQuota SpecialQuota
	err := DB.Where("user_id = ? AND channel_id = ? AND model = ?", userId, channelId, model).First(&specialQuota).Error
	if err != nil {
		return err
	}
	// 更新quota，允许负数（前端应该防止这种情况）
	return DB.Model(&specialQuota).Update("quota", gorm.Expr("quota - ?", quota)).Error
}

// SpecialQuotaInfo 用于返回给前端的专项余额信息
type SpecialQuotaInfo struct {
	ChannelId   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Model       string `json:"model"`
	Quota       int64  `json:"quota"`
}

// GetUserSpecialQuotasWithChannelInfo 获取用户的专项余额，包含渠道名称
func GetUserSpecialQuotasWithChannelInfo(userId int) ([]*SpecialQuotaInfo, error) {
	var results []*SpecialQuotaInfo
	err := DB.Table("special_quotas").
		Select("special_quotas.channel_id, special_quotas.model, special_quotas.quota, COALESCE(channels.name, '不限渠道') as channel_name").
		Joins("LEFT JOIN channels ON special_quotas.channel_id = channels.id").
		Where("special_quotas.user_id = ? AND special_quotas.quota > 0", userId).
		Order("special_quotas.id desc").
		Scan(&results).Error
	return results, err
}

// ModelList 用于存储模型列表的JSON结构
type ModelList struct {
	Models []string `json:"models"`
}

// ParseModels 解析模型列表JSON字符串
func ParseModels(modelsJSON string) ([]string, error) {
	if modelsJSON == "" {
		return []string{}, nil
	}
	var modelList ModelList
	err := json.Unmarshal([]byte(modelsJSON), &modelList)
	if err != nil {
		return nil, err
	}
	return modelList.Models, nil
}

// EncodeModels 将模型列表编码为JSON字符串
func EncodeModels(models []string) (string, error) {
	modelList := ModelList{Models: models}
	data, err := json.Marshal(modelList)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
