#!/bin/bash

categories=("Bug Fixes" "Security Fixes" "Breaking Changes" "Improvements" "New Features")
components=("WebUI" "Notebook" "TensorBoard" "Command" "Shell" "Experiment" "API" "Images")

autocomplete() {
    local items=("$@")
    select item in "${items[@]}"; do
        if [[ -n $item ]]; then
            echo "$item"
            break
        else
            echo "Invalid selection"
        fi
    done
}

echo "Enter the filename for the release note (without extension):"
read filename

echo "Choose a prefix (or enter a new one):"
prefix=$(autocomplete "${categories[@]}")

echo "Choose a title (or enter a new one):"
title=$(autocomplete "${components[@]}")

echo "Enter the component details:"
read component

cat <<EOL >"${filename}.rst"
:orphan:

**${prefix}**

-  ${title}: ${component}
EOL

echo "Release note file '${filename}.rst' has been created successfully!"
