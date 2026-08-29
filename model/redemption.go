package model

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
)

const (
	RedemptionCodeStatusEnabled  = 1 // don't use 0, 0 is the default value!
	RedemptionCodeStatusDisabled = 2 // also don't use 0
	RedemptionCodeStatusUsed     = 3 // also don't use 0
)

const (
	RedemptionTypeGeneral = 1 // 普通兑换码
	RedemptionTypeSpecial = 2 // 专用兑换码
)

type Redemption struct {
	Id           int    `json:"id"`
	UserId       int    `json:"user_id"`
	Key          string `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status       int    `json:"status" gorm:"default:1"`
	Name         string `json:"name" gorm:"index"`
	Quota        int64  `json:"quota" gorm:"bigint;default:100"`
	CreatedTime  int64  `json:"created_time" gorm:"bigint"`
	RedeemedTime int64  `json:"redeemed_time" gorm:"bigint"`
	Count        int    `json:"count" gorm:"-:all"`       // only for api request
	Type         int    `json:"type" gorm:"default:1"`     // 1=普通, 2=专用
	ChannelId    int    `json:"channel_id" gorm:"default:0"` // 专用兑换码的渠道ID，0表示不限渠道
	Models       string `json:"models" gorm:"type:text"`    // 专用兑换码的模型列表，JSON格式
	ActivityTag  string `json:"activity_tag" gorm:"index;type:varchar(50)"` // 活动标签，同一活动标签的兑换码每个用户只能兑换一个
	RedeemedUserId int  `json:"redeemed_user_id" gorm:"index"` // 兑换该码的用户ID
}

func GetAllRedemptions(startIdx int, num int) ([]*Redemption, error) {
	var redemptions []*Redemption
	var err error
	err = DB.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	return redemptions, err
}

func SearchRedemptions(keyword string) (redemptions []*Redemption, err error) {
	err = DB.Where("id = ? or name LIKE ?", keyword, keyword+"%").Find(&redemptions).Error
	return redemptions, err
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func Redeem(ctx context.Context, key string, userId int) (quota int64, err error) {
	if key == "" {
		return 0, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return 0, errors.New("无效的 user id")
	}
	redemption := &Redemption{}

	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		
		// 检查活动标签限制
		if redemption.ActivityTag != "" {
			var count int64
			err := tx.Model(&Redemption{}).Where("activity_tag = ? AND redeemed_user_id = ? AND status = ?", 
				redemption.ActivityTag, userId, RedemptionCodeStatusUsed).Count(&count).Error
			if err != nil {
				return errors.New("检查活动标签失败")
			}
			if count > 0 {
				return errors.New("您已兑换过该活动的兑换码，不能重复兑换")
			}
		}

		// 根据兑换码类型处理
		if redemption.Type == RedemptionTypeSpecial {
			// 专用兑换码，增加专项余额
			models, parseErr := ParseModels(redemption.Models)
			if parseErr != nil {
				return errors.New("兑换码模型配置错误")
			}
			if len(models) == 0 {
				return errors.New("专用兑换码未配置模型")
			}
			// 为每个模型增加专项余额
			for _, model := range models {
				err = IncreaseSpecialQuota(userId, redemption.ChannelId, model, redemption.Quota)
				if err != nil {
					return err
				}
			}
		} else {
			// 普通兑换码，增加通用余额
			err = tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
			if err != nil {
				return err
			}
		}

		redemption.RedeemedTime = helper.GetTimestamp()
		redemption.Status = RedemptionCodeStatusUsed
		redemption.RedeemedUserId = userId
		err = tx.Save(redemption).Error
		return err
	})
	if err != nil {
		return 0, errors.New("兑换失败，" + err.Error())
	}

	if redemption.Type == RedemptionTypeSpecial {
		models, _ := ParseModels(redemption.Models)
		modelStr := ""
		if len(models) > 0 {
			modelStr = fmt.Sprintf("(%s)", models[0])
			if len(models) > 1 {
				modelStr = fmt.Sprintf("(%d个模型)", len(models))
			}
		}
		RecordLog(ctx, userId, LogTypeTopup, fmt.Sprintf("通过专用兑换码充值 %s %s", common.LogQuota(redemption.Quota), modelStr))
	} else {
		RecordLog(ctx, userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s", common.LogQuota(redemption.Quota)))
	}
	return redemption.Quota, nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "redeemed_time", "type", "channel_id", "models", "activity_tag").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DisableRedemptionsByName(name string) (count int64, err error) {
	if name == "" {
		return 0, errors.New("名称为空！")
	}
	result := DB.Model(&Redemption{}).Where("name = ? AND status = ?", name, RedemptionCodeStatusEnabled).Update("status", RedemptionCodeStatusDisabled)
	return result.RowsAffected, result.Error
}
