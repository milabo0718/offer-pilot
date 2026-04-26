set -euo pipefail

ENV_FILE="${1:-.env.local}"
MODE="${2:-backend}"
INGEST_DIR="${3:-./examples/rag_data_structured_strict}"
MOCK_FLAG="${4:-false}"

trim() {
  local s="$1"
  # 去除前后空白字符
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

if [[ ! -f "$ENV_FILE" ]]; then
  echo "环境文件不存在: $ENV_FILE"
  exit 1
fi

line_no=0
while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  line_no=$((line_no + 1))
  line="$(trim "$raw_line")"

  # 跳过空行和注释
  [[ -z "$line" || "${line:0:1}" == "#" ]] && continue

  if [[ "$line" != *=* ]]; then
    echo "环境文件格式错误（第 ${line_no} 行）：缺少 '='"
    exit 1
  fi

  key="${line%%=*}"
  value="${line#*=}"
  key="$(trim "$key")"
  value="$(trim "$value")"

  if [[ ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
    echo "环境变量名非法（第 ${line_no} 行）：$key"
    exit 1
  fi

  export "$key=$value"
done < "$ENV_FILE"

if [[ "$MODE" == "backend" ]]; then
  echo "Environment loaded. Starting backend..."
  go run .
  exit $?
fi

if [[ "$MODE" == "rag-ingest" ]]; then
  echo "Environment loaded. Starting rag_ingest..."
  echo "dir=$INGEST_DIR mock=$MOCK_FLAG"
  go run ./cmd/rag_ingest -dir "$INGEST_DIR" -mock="$MOCK_FLAG"
  exit $?
fi

echo "未知模式: $MODE（可选: backend | rag-ingest）"
exit 1