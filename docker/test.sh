#!/bin/bash

# One-API Codex 功能测试脚本

echo "=========================================="
echo "  One-API Codex 功能测试"
echo "=========================================="
echo ""

# 检查参数
if [ $# -lt 2 ]; then
    echo "用法: ./test.sh <API_URL> <API_KEY>"
    echo ""
    echo "示例:"
    echo "  ./test.sh http://localhost:3000 sk-xxxxxxxxxxxx"
    echo ""
    exit 1
fi

API_URL=$1
API_KEY=$2

echo "测试配置:"
echo "  API URL: $API_URL"
echo "  API Key: ${API_KEY:0:10}..."
echo ""

# 测试 1: 基本的 Responses API 请求
echo "测试 1: Codex 格式转换（流式）"
echo "----------------------------------------"

RESPONSE=$(curl -s -N "${API_URL}/v1/responses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${API_KEY}" \
  -d '{
    "model": "gpt-3.5-turbo",
    "input_items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Say hello in one word"
          }
        ]
      }
    ],
    "stream": true,
    "max_tokens": 10
  }' 2>&1)

# 检查响应
if echo "$RESPONSE" | grep -q "response.output_item"; then
    echo "✓ 测试通过：收到正确的 Responses API 格式响应"
    echo ""
    echo "响应示例（前 500 字符）:"
    echo "$RESPONSE" | head -c 500
    echo ""
    echo "..."
elif echo "$RESPONSE" | grep -q "error"; then
    echo "✗ 测试失败：收到错误响应"
    echo "$RESPONSE"
    exit 1
else
    echo "⚠ 测试结果不确定，请检查响应:"
    echo "$RESPONSE" | head -c 500
fi

echo ""
echo "=========================================="
echo ""

# 测试 2: 验证普通 Chat Completions 请求仍然正常
echo "测试 2: 普通 Chat Completions 请求"
echo "----------------------------------------"

RESPONSE2=$(curl -s "${API_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${API_KEY}" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "Say hi in one word"
      }
    ],
    "stream": false,
    "max_tokens": 10
  }' 2>&1)

if echo "$RESPONSE2" | grep -q "choices"; then
    echo "✓ 测试通过：Chat Completions API 正常工作"
    echo ""
    echo "响应示例:"
    echo "$RESPONSE2" | head -c 300
    echo ""
elif echo "$RESPONSE2" | grep -q "error"; then
    echo "✗ 测试失败：收到错误响应"
    echo "$RESPONSE2"
else
    echo "⚠ 测试结果不确定"
    echo "$RESPONSE2" | head -c 300
fi

echo ""
echo "=========================================="
echo "  测试完成"
echo "=========================================="
echo ""
echo "说明："
echo "  - 测试 1 验证 Codex 转换功能是否正常"
echo "  - 测试 2 验证普通请求不受影响"
echo ""
echo "如果两个测试都通过，说明 Codex 功能已正确部署。"
echo ""
