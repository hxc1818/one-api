package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller/validator"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

func RelayResponsesHelper(c *gin.Context) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	
	// get & validate responses request
	responsesRequest, err := getAndValidateResponsesRequest(c)
	if err != nil {
		logger.Errorf(ctx, "getAndValidateResponsesRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_responses_request", http.StatusBadRequest)
	}
	meta.IsStream = responsesRequest.Stream

	// map model name
	meta.OriginModelName = responsesRequest.Model
	responsesRequest.Model, _ = getMappedModelName(responsesRequest.Model, meta.ModelMapping)
	meta.ActualModelName = responsesRequest.Model
	
	// get model ratio & group ratio
	modelRatio := billingratio.GetModelRatio(responsesRequest.Model, meta.ChannelType)
	groupRatio := billingratio.GetGroupRatio(meta.Group)
	ratio := modelRatio * groupRatio
	
	// pre-consume quota - estimate based on input_items
	promptTokens := getResponsesPromptTokens(responsesRequest)
	meta.PromptTokens = promptTokens
	preConsumedQuota, bizErr := preConsumeQuota(ctx, responsesRequest, promptTokens, ratio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "preConsumeQuota failed: %+v", *bizErr)
		return bizErr
	}

	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// get request body
	requestBody, err := getResponsesRequestBody(c, meta, responsesRequest, adaptor)
	if err != nil {
		return openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	// do request
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(meta, resp) {
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return RelayErrorHandler(resp)
	}

	// do response
	usage, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return respErr
	}
	
	// post-consume quota
	go postConsumeQuota(ctx, usage, meta, responsesRequest, ratio, preConsumedQuota, modelRatio, groupRatio, false)
	return nil
}

func getAndValidateResponsesRequest(c *gin.Context) (*model.GeneralOpenAIRequest, error) {
	responsesRequest := &model.GeneralOpenAIRequest{}
	err := common.UnmarshalBodyReusable(c, responsesRequest)
	if err != nil {
		return nil, err
	}
	
	err = validator.ValidateResponsesRequest(responsesRequest)
	if err != nil {
		return nil, err
	}
	return responsesRequest, nil
}

func getResponsesPromptTokens(request *model.GeneralOpenAIRequest) int {
	// Estimate tokens from input_items
	// This is a simplified estimation - actual implementation may need more sophisticated counting
	totalTokens := 0
	for _, item := range request.InputItems {
		for _, content := range item.Content {
			if content.Type == "text" && content.Text != "" {
				totalTokens += openai.CountTokenInput(content.Text, request.Model)
			}
		}
	}
	return totalTokens
}

func getResponsesRequestBody(c *gin.Context, meta *meta.Meta, responsesRequest *model.GeneralOpenAIRequest, adaptor adaptor.Adaptor) (io.Reader, error) {
	if !config.EnforceIncludeUsage &&
		meta.APIType == apitype.OpenAI &&
		meta.OriginModelName == meta.ActualModelName &&
		meta.ChannelType != channeltype.Baichuan {
		// no need to convert request for openai
		return c.Request.Body, nil
	}

	// get request body
	var requestBody io.Reader
	convertedRequest, err := adaptor.ConvertRequest(c, meta.Mode, responsesRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request failed: %s\n", err.Error())
		return nil, err
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request json_marshal_failed: %s\n", err.Error())
		return nil, err
	}
	logger.Debugf(c.Request.Context(), "converted request: \n%s", string(jsonData))
	requestBody = bytes.NewBuffer(jsonData)
	return requestBody, nil
}
