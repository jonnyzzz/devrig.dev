#!/usr/bin/env bash

export OPENAI_API_KEY="sk-"
export LITELLM_API_KEY="sk-"
export OPENAI_API_URL="http://localhost:1984/v1/chat/completions"
export LITELLM_API_URL="http://localhost:1984/v1/chat/completions"
export PRIMARY_MODEL="hetzner/openai/gpt-oss-120b"
export MATTERHORN_DEFAULT_LLM_PROVIDER="LITE_LLM"
export MATTERHORN_DEFAULT_MODEL="hetzner/openai/gpt-oss-120b"

exec ~/Applications/Junie.app/Contents/MacOS/idea
