package controller

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/relay/constant/role"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller/validator"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func getAndValidateTextRequest(c *gin.Context, relayMode int) (*relaymodel.GeneralOpenAIRequest, error) {
	textRequest := &relaymodel.GeneralOpenAIRequest{}
	err := common.UnmarshalBodyReusable(c, textRequest)
	if err != nil {
		return nil, err
	}
	
	// Handle Codex conversion: convert input_items to messages
	if c.GetBool("need_codex_conversion") && len(textRequest.InputItems) > 0 {
		// Import codex converter
		// This will be handled in the actual conversion
	}
	
	if relayMode == relaymode.Moderations && textRequest.Model == "" {
		textRequest.Model = "text-moderation-latest"
	}
	if relayMode == relaymode.Embeddings && textRequest.Model == "" {
		textRequest.Model = c.Param("model")
	}
	err = validator.ValidateTextRequest(textRequest, relayMode)
	if err != nil {
		return nil, err
	}
	return textRequest, nil
}

func getPromptTokens(textRequest *relaymodel.GeneralOpenAIRequest, relayMode int) int {
	switch relayMode {
	case relaymode.ChatCompletions:
		return openai.CountTokenMessages(textRequest.Messages, textRequest.Model)
	case relaymode.Completions:
		return openai.CountTokenInput(textRequest.Prompt, textRequest.Model)
	case relaymode.Moderations:
		return openai.CountTokenInput(textRequest.Input, textRequest.Model)
	}
	return 0
}

func getPreConsumedQuota(textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64) int64 {
	preConsumedTokens := config.PreConsumedQuota + int64(promptTokens)
	if textRequest.MaxTokens != 0 {
		preConsumedTokens += int64(textRequest.MaxTokens)
	}
	return int64(float64(preConsumedTokens) * ratio)
}

func preConsumeQuota(ctx context.Context, textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64, meta *meta.Meta) (int64, *relaymodel.ErrorWithStatusCode) {
	preConsumedQuota := getPreConsumedQuota(textRequest, promptTokens, ratio)

	// 先检查是否有专项余额
	specialQuota, err := model.GetUserSpecialQuota(meta.UserId, meta.ChannelId, textRequest.Model)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "get_special_quota_failed", http.StatusInternalServerError)
	}

	// 如果有专项余额且足够，优先使用专项余额
	if specialQuota >= preConsumedQuota {
		err = model.DecreaseSpecialQuota(meta.UserId, meta.ChannelId, textRequest.Model, preConsumedQuota)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "decrease_special_quota_failed", http.StatusInternalServerError)
		}
		// 标记使用了专项余额
		meta.UseSpecialQuota = true
		meta.SpecialQuotaUsed = preConsumedQuota
		logger.Info(ctx, fmt.Sprintf("user %d uses special quota %d for model %s", meta.UserId, preConsumedQuota, textRequest.Model))
		return preConsumedQuota, nil
	}

	// 如果有部分专项余额，先消耗专项余额，剩余部分消耗通用余额
	if specialQuota > 0 {
		err = model.DecreaseSpecialQuota(meta.UserId, meta.ChannelId, textRequest.Model, specialQuota)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "decrease_special_quota_failed", http.StatusInternalServerError)
		}
		meta.UseSpecialQuota = true
		meta.SpecialQuotaUsed = specialQuota
		logger.Info(ctx, fmt.Sprintf("user %d uses partial special quota %d for model %s", meta.UserId, specialQuota, textRequest.Model))
		// 剩余部分从通用余额扣除
		remainingQuota := preConsumedQuota - specialQuota
		userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
		}
		if userQuota-remainingQuota < 0 {
			// 通用余额不足，需要回滚专项余额
			_ = model.IncreaseSpecialQuota(meta.UserId, meta.ChannelId, textRequest.Model, specialQuota)
			return preConsumedQuota, openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
		}
		err = model.CacheDecreaseUserQuota(meta.UserId, remainingQuota)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
		}
		if userQuota > 100*preConsumedQuota {
			preConsumedQuota = 0
			logger.Info(ctx, fmt.Sprintf("user %d has enough quota %d, trusted and no need to pre-consume", meta.UserId, userQuota))
		}
		if preConsumedQuota > 0 {
			err := model.PreConsumeTokenQuota(meta.TokenId, preConsumedQuota)
			if err != nil {
				return preConsumedQuota, openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
			}
		}
		return preConsumedQuota, nil
	}

	// 没有专项余额，使用通用余额
	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-preConsumedQuota < 0 {
		return preConsumedQuota, openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	err = model.CacheDecreaseUserQuota(meta.UserId, preConsumedQuota)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota > 100*preConsumedQuota {
		// in this case, we do not pre-consume quota
		// because the user has enough quota
		preConsumedQuota = 0
		logger.Info(ctx, fmt.Sprintf("user %d has enough quota %d, trusted and no need to pre-consume", meta.UserId, userQuota))
	}
	if preConsumedQuota > 0 {
		err := model.PreConsumeTokenQuota(meta.TokenId, preConsumedQuota)
		if err != nil {
			return preConsumedQuota, openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
	}
	return preConsumedQuota, nil
}

func postConsumeQuota(ctx context.Context, usage *relaymodel.Usage, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, ratio float64, preConsumedQuota int64, modelRatio float64, groupRatio float64, systemPromptReset bool) {
	if usage == nil {
		logger.Error(ctx, "usage is nil, which is unexpected")
		return
	}
	var quota int64
	completionRatio := billingratio.GetCompletionRatio(textRequest.Model, meta.ChannelType)
	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens
	quota = int64(math.Ceil((float64(promptTokens) + float64(completionTokens)*completionRatio) * ratio))
	if ratio != 0 && quota <= 0 {
		quota = 1
	}
	totalTokens := promptTokens + completionTokens
	if totalTokens == 0 {
		// in this case, must be some error happened
		// we cannot just return, because we may have to return the pre-consumed quota
		quota = 0
	}

	// 处理专项余额的情况
	if meta.UseSpecialQuota {
		// 已经使用了专项余额
		if quota <= meta.SpecialQuotaUsed {
			// 实际消耗小于等于预消耗的专项余额，需要退还多余的专项余额
			refund := meta.SpecialQuotaUsed - quota
			if refund > 0 {
				err := model.IncreaseSpecialQuota(meta.UserId, meta.ChannelId, textRequest.Model, refund)
				if err != nil {
					logger.Error(ctx, "error refunding special quota: "+err.Error())
				}
			}
			// 如果之前还消耗了部分通用余额，也需要退还
			if preConsumedQuota > meta.SpecialQuotaUsed {
				_ = preConsumedQuota - meta.SpecialQuotaUsed
				err := model.CacheUpdateUserQuota(ctx, meta.UserId)
				if err != nil {
					logger.Error(ctx, "error update user quota cache: "+err.Error())
				}
			}
		} else {
			// 实际消耗大于预消耗的专项余额，需要额外扣除通用余额
			additionalQuota := quota - meta.SpecialQuotaUsed
			quotaDelta := additionalQuota - (preConsumedQuota - meta.SpecialQuotaUsed)
			err := model.PostConsumeTokenQuota(meta.TokenId, quotaDelta)
			if err != nil {
				logger.Error(ctx, "error consuming token remain quota: "+err.Error())
			}
			err = model.CacheUpdateUserQuota(ctx, meta.UserId)
			if err != nil {
				logger.Error(ctx, "error update user quota cache: "+err.Error())
			}
		}
		logContent := fmt.Sprintf("专项余额：%s | 倍率：%.2f × %.2f × %.2f", common.LogQuota(meta.SpecialQuotaUsed), modelRatio, groupRatio, completionRatio)
		model.RecordConsumeLog(ctx, &model.Log{
			UserId:            meta.UserId,
			ChannelId:         meta.ChannelId,
			PromptTokens:      promptTokens,
			CompletionTokens:  completionTokens,
			ModelName:         textRequest.Model,
			TokenName:         meta.TokenName,
			Quota:             int(quota),
			Content:           logContent,
			IsStream:          meta.IsStream,
			ElapsedTime:       helper.CalcElapsedTime(meta.StartTime),
			SystemPromptReset: systemPromptReset,
		})
		model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
		model.UpdateChannelUsedQuota(meta.ChannelId, quota)
		return
	}

	// 未使用专项余额，按原逻辑处理
	quotaDelta := quota - preConsumedQuota
	err := model.PostConsumeTokenQuota(meta.TokenId, quotaDelta)
	if err != nil {
		logger.Error(ctx, "error consuming token remain quota: "+err.Error())
	}
	err = model.CacheUpdateUserQuota(ctx, meta.UserId)
	if err != nil {
		logger.Error(ctx, "error update user quota cache: "+err.Error())
	}
	logContent := fmt.Sprintf("倍率：%.2f × %.2f × %.2f", modelRatio, groupRatio, completionRatio)
	model.RecordConsumeLog(ctx, &model.Log{
		UserId:            meta.UserId,
		ChannelId:         meta.ChannelId,
		PromptTokens:      promptTokens,
		CompletionTokens:  completionTokens,
		ModelName:         textRequest.Model,
		TokenName:         meta.TokenName,
		Quota:             int(quota),
		Content:           logContent,
		IsStream:          meta.IsStream,
		ElapsedTime:       helper.CalcElapsedTime(meta.StartTime),
		SystemPromptReset: systemPromptReset,
	})
	model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
	model.UpdateChannelUsedQuota(meta.ChannelId, quota)
}

func getMappedModelName(modelName string, mapping map[string]string) (string, bool) {
	if mapping == nil {
		return modelName, false
	}
	mappedModelName := mapping[modelName]
	if mappedModelName != "" {
		return mappedModelName, true
	}
	return modelName, false
}

func isErrorHappened(meta *meta.Meta, resp *http.Response) bool {
	if resp == nil {
		if meta.ChannelType == channeltype.AwsClaude {
			return false
		}
		return true
	}
	if resp.StatusCode != http.StatusOK &&
		// replicate return 201 to create a task
		resp.StatusCode != http.StatusCreated {
		return true
	}
	if meta.ChannelType == channeltype.DeepL {
		// skip stream check for deepl
		return false
	}

	if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") &&
		// Even if stream mode is enabled, replicate will first return a task info in JSON format,
		// requiring the client to request the stream endpoint in the task info
		meta.ChannelType != channeltype.Replicate {
		return true
	}
	return false
}

func setSystemPrompt(ctx context.Context, request *relaymodel.GeneralOpenAIRequest, prompt string) (reset bool) {
	if prompt == "" {
		return false
	}
	if len(request.Messages) == 0 {
		return false
	}
	if request.Messages[0].Role == role.System {
		request.Messages[0].Content = prompt
		logger.Infof(ctx, "rewrite system prompt")
		return true
	}
	request.Messages = append([]relaymodel.Message{{
		Role:    role.System,
		Content: prompt,
	}}, request.Messages...)
	logger.Infof(ctx, "add system prompt")
	return true
}
