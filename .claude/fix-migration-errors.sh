#!/bin/bash

# 修复错误码迁移后的编译错误
# 手动修复那些参数格式不正确的函数调用

set -e

GOAGENT_DIR="/home/hellotalk/code/go/src/github.com/kart-io/sentinel-x/staging/src/github.com/kart-io/goagent"

echo "修复错误码迁移后的问题..."

# 修复 store/langgraph_store.go
sed -i '174s/.*/\t\treturn nil, agentErrors.NewError(agentErrors.CodeNotFound, fmt.Sprintf("key not found: %s in namespace %v", key, namespace))./' \
  "$GOAGENT_DIR/store/langgraph_store.go"

sed -i '262s/.*/\t\treturn agentErrors.NewError(agentErrors.CodeNotFound, fmt.Sprintf("key not found: %s in namespace %v", key, namespace))/' \
  "$GOAGENT_DIR/store/langgraph_store.go"

# 修复 llm/factory.go
sed -i '75s/.*/\t\treturn agentErrors.NewError(agentErrors.CodeAgentConfig, "config is nil").WithComponent("llm_factory").WithContext("field", "config")/' \
  "$GOAGENT_DIR/llm/factory.go"

sed -i '80s/.*/\t\treturn agentErrors.NewError(agentErrors.CodeAgentConfig, "provider is required").WithComponent("llm_factory").WithContext("field", "provider")/' \
  "$GOAGENT_DIR/llm/factory.go"

sed -i '100s/.*/\t\treturn agentErrors.NewError(agentErrors.CodeAgentConfig, fmt.Sprintf("API key is required for provider: %s", opts.Provider)).WithComponent("llm_factory").WithContext("provider", string(opts.Provider))/' \
  "$GOAGENT_DIR/llm/factory.go"

# 修复 examples/error_handling/main.go
sed -i '17s/.*/\t\terr := errors.NewErrorWithCause(errors.CodeExternalService, "API request failed", fmt.Errorf("connection timeout"))./' \
  "$GOAGENT_DIR/examples/error_handling/main.go"

sed -i '23s/.*/\tnestedErr := errors.NewErrorWithCause(errors.CodeToolExecution, "tool execution failed", apiErr)./' \
  "$GOAGENT_DIR/examples/error_handling/main.go"

sed -i '28s/.*/\tif errors.IsCode(nestedErr, errors.CodeNotFound) {/' \
  "$GOAGENT_DIR/examples/error_handling/main.go"

sed -i '33s/.*/\twrappedErr := errors.NewErrorWithCause(errors.CodeAgentExecution, "agent task failed", nestedErr)./' \
  "$GOAGENT_DIR/examples/error_handling/main.go"

sed -i '87s/.*/\tif errors.IsCode(err, errors.CodeNotFound) {/' \
  "$GOAGENT_DIR/examples/error_handling/main.go"

sed -i '88s/.*/\terr = errors.NewError(errors.CodeRateLimit, "rate limit exceeded").WithContext("retry_after", 60)/' \
  "$GOAGENT_DIR/examples/error_handling/main.go"

# 修复 core/checkpoint/distributed.go
sed -i '115s/.*/\t\treturn agentErrors.NewError(agentErrors.CodeAgentConfig, fmt.Sprintf("checkpoint manager not found for type: %s", nodeID)).WithComponent("distributed_checkpointer").WithContext("node_id", nodeID)/' \
  "$GOAGENT_DIR/core/checkpoint/distributed.go"

# 修复 core/checkpoint/redis.go
sed -i '134s/.*/\t\treturn agentErrors.NewErrorWithCause(agentErrors.CodeNetwork, "redis connection failed", err)./' \
  "$GOAGENT_DIR/core/checkpoint/redis.go"

echo "修复完成！"
