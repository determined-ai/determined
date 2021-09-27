#!/bin/bash
# For determined version 0.16.x ???

declare -A PACKAGE_MAP

GET_FILE_PATH_PYTHON="import %s; print(%s.__file__);"

# [<file_name_in_model_dir>]=<python_include_path>
PACKAGE_MAP[tensorboard_debug.py]="determined.exec.tensorboard"
# ^ Append more packages as needed.


# Figure out the file paths to each of the packages:
for pfile in "${!PACKAGE_MAP[@]}"
do
    printf -v GET_PATH "$GET_FILE_PATH_PYTHON" \
        ${PACKAGE_MAP[$pfile]} ${PACKAGE_MAP[$pfile]}
    CUR_PATH=$(python3 -c "$GET_PATH") || echo ""
    if [ -n "$CUR_PATH" ]; then
        # Copy file over if we successfully found the path
        cp -v "$pfile" "$CUR_PATH"
    else
        echo "Couldn't find file path for ${PACKAGE_MAP[$pfile]}"
    fi
done
