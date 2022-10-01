#!/bin/sh

D=webui/react
target=$1
files=$(realpath --relative-to="$D" "${@: 2}" | tr "\n" " ")

case $target in
  js    )  make -j$(nproc) -C "$D" prettier PRE_ARGS="-- -c $files" eslint ES_ARGS="$files"    ;;
  css   )  make -j$(nproc) -C "$D" prettier PRE_ARGS="-- -c $files" stylelint ST_ARGS="$files" ;;
  misc  )  make -j$(nproc) -C "$D" prettier PRE_ARGS="-- -c $files" check-package-lock         ;;
  *     )  echo "unknonwn target '$target'"; exit 1 ;;

esac
