set -euo pipefail

ENV_FILE="${1:-.env.local}"

set -a
while IFS= read -r line; do
  [[ -z "${line// }" || "${line:0:1}" == "#" ]] && continue
  export "$line"
done < "$ENV_FILE"
set +a

echo "Environment loaded. Starting backend..."
go run .