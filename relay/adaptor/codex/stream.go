package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
)

const (
	dataPrefix       = "data: "
	done             = "[DONE]"
	dataPrefixLength = len(dataPrefix)
)

// StreamHandlerWithCodexConversion converts Chat Completions stream to Responses API format
func StreamHandlerWithCodexConversion(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, string, *model.Usage) {
	responseText := ""
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	var usage *model.Usage

	common.SetEventStreamHeaders(c)

	// Generate IDs for Responses API
	// conversationID := fmt.Sprintf("conv_%d", time.Now().UnixNano())
	responseID := fmt.Sprintf("resp_%d", time.Now().UnixNano())
	outputItemID := fmt.Sprintf("item_%d", time.Now().UnixNano())
	
	firstChunk := true
	doneRendered := false

	for scanner.Scan() {
		data := scanner.Text()
		if len(data) < dataPrefixLength {
			continue
		}
		if data[:dataPrefixLength] != dataPrefix && !strings.HasPrefix(data, done) {
			continue
		}
		if strings.HasPrefix(data[dataPrefixLength:], done) {
			// Send [DONE] in Responses API format
			c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.(http.Flusher).Flush()
			doneRendered = true
			continue
		}

		// Parse chat completion stream response
		var chatResp struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			Model   string `json:"model"`
			Choices []struct {
				Index        int           `json:"index"`
				Delta        model.Message `json:"delta"`
				FinishReason *string       `json:"finish_reason"`
			} `json:"choices"`
			Usage *model.Usage `json:"usage,omitempty"`
		}

		err := json.Unmarshal([]byte(data[dataPrefixLength:]), &chatResp)
		if err != nil {
			logger.SysError("error unmarshalling stream response: " + err.Error())
			continue
		}

		if len(chatResp.Choices) == 0 && chatResp.Usage == nil {
			continue
		}

		for _, choice := range chatResp.Choices {
			// Send response.output_item.added event on first chunk with role
			if firstChunk && choice.Delta.Role != "" {
				outputItem := ResponsesOutputItem{
					ID:     outputItemID,
					Type:   "message",
					Role:   "assistant",
					Status: "in_progress",
					Content: []ResponsesContentItem{
						{
							Type: "text",
							Text: "",
						},
					},
				}

				eventData := map[string]interface{}{
					"type":         "response.output_item.added",
					"response_id":  responseID,
					"output_index": choice.Index,
					"item":         outputItem,
				}
				jsonData, _ := json.Marshal(eventData)
				c.Writer.Write([]byte(fmt.Sprintf("event: response.output_item.added\ndata: %s\n\n", string(jsonData))))
				c.Writer.(http.Flusher).Flush()
				firstChunk = false
			}

			// Extract content
			content := ""
			if choice.Delta.Content != nil {
				switch v := choice.Delta.Content.(type) {
				case string:
					content = v
				}
			}

			if content != "" {
				responseText += content

				// Send response.output_item.delta event
				delta := ResponsesDelta{
					Type:         "content_delta",
					ContentIndex: 0,
					Delta: ResponsesContentDelta{
						Type: "text_delta",
						Text: content,
					},
				}

				eventData := map[string]interface{}{
					"type":         "response.output_item.delta",
					"response_id":  responseID,
					"output_index": choice.Index,
					"item_id":      outputItemID,
					"delta":        delta,
				}
				jsonData, _ := json.Marshal(eventData)
				c.Writer.Write([]byte(fmt.Sprintf("event: response.output_item.delta\ndata: %s\n\n", string(jsonData))))
				c.Writer.(http.Flusher).Flush()
			}

			// Check if finished
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				// Send response.output_item.done event
				doneEventData := map[string]interface{}{
					"type":         "response.output_item.done",
					"response_id":  responseID,
					"output_index": choice.Index,
					"item_id":      outputItemID,
				}
				jsonData, _ := json.Marshal(doneEventData)
				c.Writer.Write([]byte(fmt.Sprintf("event: response.output_item.done\ndata: %s\n\n", string(jsonData))))
				c.Writer.(http.Flusher).Flush()

				// Send response.done event
				responseDoneData := map[string]interface{}{
					"type":        "response.done",
					"response_id": responseID,
					"status":      "completed",
				}
				jsonData, _ = json.Marshal(responseDoneData)
				c.Writer.Write([]byte(fmt.Sprintf("event: response.done\ndata: %s\n\n", string(jsonData))))
				c.Writer.(http.Flusher).Flush()
			}
		}

		// Handle usage if present
		if chatResp.Usage != nil {
			usage = chatResp.Usage
			usageEventData := map[string]interface{}{
				"type":  "response.usage",
				"usage": chatResp.Usage,
			}
			jsonData, _ := json.Marshal(usageEventData)
			c.Writer.Write([]byte(fmt.Sprintf("event: response.usage\ndata: %s\n\n", string(jsonData))))
			c.Writer.(http.Flusher).Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		logger.SysError("error reading stream: " + err.Error())
	}

	if !doneRendered {
		c.Writer.Write([]byte("data: [DONE]\n\n"))
		c.Writer.(http.Flusher).Flush()
	}

	err := resp.Body.Close()
	if err != nil {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: err.Error(),
				Type:    "one_api_error",
				Code:    "close_response_body_failed",
			},
			StatusCode: http.StatusInternalServerError,
		}, "", nil
	}

	return nil, responseText, usage
}

// ShouldUseCodexConversion checks if the request needs Codex format conversion
func ShouldUseCodexConversion(c *gin.Context) bool {
	needConversion, exists := c.Get(ctxkey.NeedCodexConversion)
	if !exists {
		return false
	}
	return needConversion.(bool)
}
