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

echo "Choose a category (or enter a new one):"
category=$(autocomplete "${categories[@]}")

echo ":orphan:" >"${filename}.rst"
echo "" >>"${filename}.rst"
echo "**${category}**" >>"${filename}.rst"
echo "" >>"${filename}.rst"

while true; do
    echo "Choose a component (or enter a new one):"
    component=$(autocomplete "${components[@]}")

    echo "Enter the details:"
    read details

    echo "-  ${component}: ${details}" >>"${filename}.rst"

    echo "Do you want to add another entry? (yes/no)"
    read answer
    if [[ $answer != "yes" ]]; then
        break
    fi
done

echo "Release note file '${filename}.rst' has been created successfully!"
