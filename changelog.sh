#!/bin/bash

set -e

## USAGE
# ./changelog.sh
# or
# ./changelog.sh WITHINSTALL

startTag=$(git tag --list --sort=version:refname | tail -n 1)
lastTag=$(git tag --list --sort=version:refname | tail -n 2 | head -n 1)

# startTag=HEAD
# lastTag=72f04e7

if [ -z "$startTag" ]; then
    echo "No tags found. Please create a tag first."
    exit 1
fi

if [ -z "$lastTag" ]; then
    echo "Only one tag found: $startTag. No changes to show."
    exit 0
fi

# if script argument of "WITHINSTALL" is present
if [ "$1" == "WITHINSTALL" ]; then
    cat <<EOF
## Install

### macOS (x86_64)

1. Download sq-darwin-x86_64.tar.gz
2. Run xattr -c ./sq-darwin-x86_64.tar.gz (to avoid "unknown developer" warning)
3. Extract: tar xzvf sq-darwin-x86_64.tar.gz
4. Run ./sq-darwin-x86_64/bin/sq

### macOS (arm64)

1. Download sq-darwin-arm64.tar.gz
2. Run xattr -c ./sq-darwin-arm64.tar.gz (to avoid "unknown developer" warning)
3. Extract: tar xzvf sq-darwin-arm64.tar.gz
4. Run ./sq-darwin-arm64/bin/sq

### Linux (x86_64)

1. Download sq-linux-x86_64.tar.gz
2. Extract: tar xzvf sq-linux-x86_64.tar.gz
3. Run ./sq-linux-x86_64/bin/sq

EOF
fi

date=$(git log -1 --format=%cd --date=short "$startTag")

echo "## [${startTag}](https://github.com/chriserin/sq/compare/${lastTag}...${startTag}) ($date)"
echo ""

features=$(git log --oneline "${startTag}"..."${lastTag}" | awk ' $2 ~ /^feat/ {print}')

if [ -n "$features" ]; then
    echo "### Features"
    echo ""
    while IFS= read -r line; do
        commit_hash=$(echo "$line" | awk '{print $1}')
        commit_type=$(echo "$line" | awk '{print $2}')
        commit_message=$(echo "$line" | cut -d' ' -f3-)

        # Extract scope if present (e.g., feat(system) -> system)
        regex='feat\(([^)]+)\)'
        commit_type=$(echo "$line" | awk '{print $2}')
        if [[ $commit_type =~ $regex ]]; then
            scope="${BASH_REMATCH[1]}"
            commit_message="${scope}: ${commit_message}"
        fi

        echo "* $commit_message [${commit_hash}](https://github.com/chriserin/sq/commit/${commit_hash}) "
    done <<<"$features"
else
    echo "### Features"
    echo ""
    echo "No new features."
    echo ""
fi

fixes=$(git log --oneline "${startTag}"..."${lastTag}" | awk ' $2 ~ /^fix/ {print}')

if [ -n "$fixes" ]; then
    echo ""
    echo "### Fixes"
    echo ""
    while IFS= read -r line; do
        commit_hash=$(echo "$line" | awk '{print $1}')
        commit_type=$(echo "$line" | awk '{print $2}')
        commit_message=$(echo "$line" | cut -d' ' -f3-)

        # Extract scope if present (e.g., fix(system) -> system)
        regex='fix\(([^)]+)\)'
        if [[ $commit_type =~ $regex ]]; then
            scope="${BASH_REMATCH[1]}"
            commit_message="${scope}: ${commit_message}"
        fi

        echo "* $commit_message [${commit_hash}](https://github.com/chriserin/sq/commit/${commit_hash}) "
    done <<<"$fixes"
else
    echo "### Fixes"
    echo ""
    echo "No bug fixes."
    echo ""
fi
