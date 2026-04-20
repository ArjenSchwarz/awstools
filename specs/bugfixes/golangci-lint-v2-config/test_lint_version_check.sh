#!/usr/bin/env bash
# Regression test for T-871: Make lint fails with golangci-lint v1 against v2 config.
#
# Verifies that `make lint` detects golangci-lint version mismatches and reports
# a clear, actionable error before invoking the binary.
#
# Test cases:
#   1. golangci-lint missing -> must fail with install guidance.
#   2. golangci-lint v1 present -> must fail with v2 required guidance.
#   3. golangci-lint v2 present -> must invoke `golangci-lint run`.
#
# Run from the repository root:
#   bash specs/bugfixes/golangci-lint-v2-config/test_lint_version_check.sh

set -u

repo_root="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$repo_root" || exit 99

fail_count=0
pass_count=0

report() {
    local name="$1"
    local ok="$2"
    if [ "$ok" = "1" ]; then
        echo "PASS: $name"
        pass_count=$((pass_count + 1))
    else
        echo "FAIL: $name"
        fail_count=$((fail_count + 1))
    fi
}

run_case() {
    local name="$1"
    local path_override="$2"
    local expect_exit="$3"
    local expect_substr="$4"

    local output status
    output="$(PATH="$path_override" make lint 2>&1)"
    status=$?

    local ok=1
    if [ "$expect_exit" = "0" ]; then
        if [ "$status" -ne 0 ]; then
            ok=0
            echo "  expected exit 0, got $status"
            echo "  output: $output"
        fi
    else
        if [ "$status" -eq 0 ]; then
            ok=0
            echo "  expected non-zero exit, got 0"
            echo "  output: $output"
        fi
    fi

    if [ -n "$expect_substr" ]; then
        if ! printf '%s' "$output" | grep -qF "$expect_substr"; then
            ok=0
            echo "  expected output to contain: $expect_substr"
            echo "  actual output: $output"
        fi
    fi

    report "$name" "$ok"
}

# Create isolated fake bin directories for each scenario.
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Ensure core utilities (sh, sed, grep, make, etc.) remain reachable via a
# minimal base PATH composed of the directories they actually live in.
base_path=""
for tool in sh bash sed grep make mktemp cat printf; do
    dir="$(command -v "$tool" 2>/dev/null)" || continue
    dir="$(dirname "$dir")"
    case ":$base_path:" in
        *":$dir:"*) ;;
        *) base_path="${base_path:+$base_path:}$dir" ;;
    esac
done

# Scenario 1: golangci-lint missing entirely.
missing_bin="$tmpdir/missing"
mkdir -p "$missing_bin"
run_case "missing golangci-lint fails with install guidance" \
    "$missing_bin:$base_path" \
    "1" \
    "golangci-lint is not installed"

# Scenario 2: v1 fake binary on PATH.
v1_bin="$tmpdir/v1"
mkdir -p "$v1_bin"
cat > "$v1_bin/golangci-lint" <<'EOF'
#!/usr/bin/env bash
case "$1" in
    --version|version)
        echo "golangci-lint has version 1.64.8 built with go1.23 from abcdef on 2025-01-01"
        exit 0
        ;;
    *)
        echo "Error: you are using a configuration file for golangci-lint v2 with golangci-lint v1: please use golangci-lint v2" 1>&2
        exit 7
        ;;
esac
EOF
chmod +x "$v1_bin/golangci-lint"
run_case "v1 golangci-lint fails with v2 required message" \
    "$v1_bin:$base_path" \
    "1" \
    "v2 is required"

# Scenario 3: v2 fake binary on PATH - the check should pass and proceed to run.
v2_bin="$tmpdir/v2"
mkdir -p "$v2_bin"
cat > "$v2_bin/golangci-lint" <<'EOF'
#!/usr/bin/env bash
case "$1" in
    --version)
        echo "golangci-lint has version 2.11.4 built with go1.25.5 from abcdef on 2025-10-01"
        exit 0
        ;;
    version)
        # v2 supports `version --short` etc., but we don't rely on it here.
        echo "golangci-lint has version 2.11.4"
        exit 0
        ;;
    run)
        # Simulate a successful lint run so the Makefile target reports success.
        echo "FAKE: golangci-lint run invoked with args: $*"
        exit 0
        ;;
    *)
        exit 0
        ;;
esac
EOF
chmod +x "$v2_bin/golangci-lint"
run_case "v2 golangci-lint passes version check and runs" \
    "$v2_bin:$base_path" \
    "0" \
    "FAKE: golangci-lint run invoked"

echo
echo "Summary: $pass_count passed, $fail_count failed."
if [ "$fail_count" -gt 0 ]; then
    exit 1
fi
exit 0
