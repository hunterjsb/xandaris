#!/bin/bash

set -e

README_PATH="entities/README.md"
START_MARKER="<!-- BEGIN CORE COMPONENTS -->"
END_MARKER="<!-- END CORE COMPONENTS -->"

# Create a temporary file to hold the new content
CONTENT_FILE=$(mktemp)

# Use a subshell to generate the content
{
    echo '- **`types.go`**: Core interfaces and base types for all entities.'
    echo '- **`registry.go`**: The generator registry system for procedural generation.'
    find entities -maxdepth 1 -name "*.go" -printf "%f\n" | sort | while read -r FILE; do
        if [[ "$FILE" != "types.go" && "$FILE" != "registry.go" ]]; then
            DESCRIPTION=$(grep -m 1 '^//' "entities/$FILE" | sed 's|^//[ \t]*||' || echo "No description comment found.")
            # Use printf for safer output and to avoid issues with backticks in echo
            printf -- '- **`%s`**: %s\n' "$FILE" "$DESCRIPTION"
        fi
    done
} > "$CONTENT_FILE"

# Create a temporary README file
TMP_README=$(mktemp)

# Replace the content in the README
awk -v content_file="$CONTENT_FILE" '
    /<!-- BEGIN CORE COMPONENTS -->/ {
        print
        while ((getline line < content_file) > 0) {
            print line
        }
        f = 1
    }
    /<!-- END CORE COMPONENTS -->/ {
        f = 0
    }
    !f {
        print
    }
' "$README_PATH" > "$TMP_README"

# Check if the file has changed
if ! cmp -s "$TMP_README" "$README_PATH"; then
    echo "README.md is out of date. Updating..."
    mv "$TMP_README" "$README_PATH"
else
    echo "README.md is up to date."
    rm "$TMP_README"
fi

rm "$CONTENT_FILE"
