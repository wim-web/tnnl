#!/usr/bin/env bash
set -euo pipefail

readme="${1:-README.md}"
required=(
  "## Quickstart"
  "session-manager-plugin --version"
  "aws-vault exec"
  "AWS_PROFILE"
  "AWS_REGION"
  "aws configure list"
  "session-manager-prerequisites.html"
  "list_ecs.html"
  "PATH_FROM_ERROR"
  "explicit CLI flag > input file > default"
  "ecs:ListClusters"
  "ecs:ListTasks"
  "ecs:DescribeTasks"
  "ecs:ExecuteCommand"
  "ssm:StartSession"
  "ssm:TerminateSession"
  "ssmmessages:CreateControlChannel"
  "ssmmessages:CreateDataChannel"
  "ssmmessages:OpenControlChannel"
  "ssmmessages:OpenDataChannel"
  "ExecuteCommandAgent"
  "runtime ID"
  "tnnl exec --wait"
  "checksum mismatch"
  "candidate version mismatch"
)

for value in "${required[@]}"; do
  grep -Fq -- "$value" "$readme" || {
    echo "$readme is missing required guidance: $value" >&2
    exit 1
  }
done
