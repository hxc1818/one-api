package codex

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/relay/model"
)

// InputItemsToMessages converts Responses API input_items to Chat Completions messages
func InputItemsToMessages(inputItems []model.InputItem) []model.Message {
	messages := make([]model.Message, 0, len(inputItems))
	
	for _, item := range inputItems {
		if item.Type != "message" {
			continue
		}
		
		message := model.Message{
			Role: item.Role,
		}
		
		// Convert content array to message content
		if len(item.Content) == 1 && item.Content[0].Type == "text" {
			// Simple text content
			message.Content = item.Content[0].Text
		} else {
			// Multimodal content - keep as array
			contentArray := make([]model.MessageContent, 0, len(item.Content))
			for _, content := range item.Content {
				mc := model.MessageContent{
					Type: content.Type,
				}
				if content.Type == "text" {
					mc.Text = content.Text
				} else if content.Type == "image_url" && content.ImageUrl != nil {
					mc.ImageURL = &model.MessageImageURL{
						Url:    content.ImageUrl.Url,
						Detail: content.ImageUrl.Detail,
					}
				}
				contentArray = append(contentArray, mc)
			}
			message.Content = contentArray
		}
		
		messages = append(messages, message)
	}
	
	return messages
}

// ResponsesStreamEvent represents a Server-Sent Event for Responses API
type ResponsesStreamEvent struct {
	Event string          `json:"-"` // event type, sent as "event: xxx"
	Data  json.RawMessage `json:"-"` // data payload, sent as "data: {...}"
}

// ResponsesOutputItem represents an output item in Responses API
type ResponsesOutputItem struct {
	ID          string                    `json:"id"`
	Type        string                    `json:"type"` // "message"
	Role        string                    `json:"role"` // "assistant"
	Status      string                    `json:"status,omitempty"` // "in_progress", "completed"
	Content     []ResponsesContentItem    `json:"content,omitempty"`
}

// ResponsesContentItem represents content in a responses output item
type ResponsesContentItem struct {
	Type string `json:"type"` // "text"
	Text string `json:"text,omitempty"`
}

// ResponsesDelta represents a delta update in Responses API
type ResponsesDelta struct {
	Type        string                    `json:"type"` // "content_delta"
	ContentIndex int                      `json:"content_index"`
	Delta       ResponsesContentDelta     `json:"delta"`
}

// ResponsesContentDelta represents the delta content
type ResponsesContentDelta struct {
	Type string `json:"type"` // "text_delta"
	Text string `json:"text,omitempty"`
}

// ChatStreamToResponsesStream converts Chat Completions stream to Responses API stream
func ChatStreamToResponsesStream(chatData string, conversationID string, responseID string, outputItemID string) []string {
	var events []string
	
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
	
	err := json.Unmarshal([]byte(chatData), &chatResp)
	if err != nil {
		return events
	}
	
	// If no conversation ID provided, use chat response ID
	if conversationID == "" {
		conversationID = chatResp.ID
	}
	if responseID == "" {
		responseID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	if outputItemID == "" {
		outputItemID = fmt.Sprintf("item_%d", time.Now().UnixNano())
	}
	
	for _, choice := range chatResp.Choices {
		// Check if this is the first chunk (has role)
		if choice.Delta.Role != "" {
			// Send output_item.added event
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
			
			data, _ := json.Marshal(map[string]interface{}{
				"type":      "response.output_item.added",
				"response_id": responseID,
				"output_index": choice.Index,
				"item":      outputItem,
			})
			events = append(events, fmt.Sprintf("event: response.output_item.added\ndata: %s\n", string(data)))
		}
		
		// Check if has content delta
		content := ""
		if choice.Delta.Content != nil {
			switch v := choice.Delta.Content.(type) {
			case string:
				content = v
			}
		}
		
		if content != "" {
			// Send output_item.delta event
			delta := ResponsesDelta{
				Type:         "content_delta",
				ContentIndex: 0,
				Delta: ResponsesContentDelta{
					Type: "text_delta",
					Text: content,
				},
			}
			
			data, _ := json.Marshal(map[string]interface{}{
				"type":         "response.output_item.delta",
				"response_id":  responseID,
				"output_index": choice.Index,
				"item_id":      outputItemID,
				"delta":        delta,
			})
			events = append(events, fmt.Sprintf("event: response.output_item.delta\ndata: %s\n", string(data)))
		}
		
		// Check if finished
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			// Send output_item.done event
			data, _ := json.Marshal(map[string]interface{}{
				"type":         "response.output_item.done",
				"response_id":  responseID,
				"output_index": choice.Index,
				"item_id":      outputItemID,
			})
			events = append(events, fmt.Sprintf("event: response.output_item.done\ndata: %s\n", string(data)))
			
			// Send response.done event
			doneData, _ := json.Marshal(map[string]interface{}{
				"type":        "response.done",
				"response_id": responseID,
				"status":      "completed",
			})
			events = append(events, fmt.Sprintf("event: response.done\ndata: %s\n", string(doneData)))
		}
	}
	
	// Handle usage if present
	if chatResp.Usage != nil {
		usageData, _ := json.Marshal(map[string]interface{}{
			"type":  "response.usage",
			"usage": chatResp.Usage,
		})
		events = append(events, fmt.Sprintf("event: response.usage\ndata: %s\n", string(usageData)))
	}
	
	return events
}
