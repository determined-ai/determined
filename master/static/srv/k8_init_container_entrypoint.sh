#!/bin/bash

# Script takes args: NUM_FILES SRC DST and un-tars files from SRC to DST.

if [ $1 -le 0 ]
then
    exit 0
fi

for var in $(seq 1 1 $1)
do
    IDX=$(( $var - 1 ))
    SRC=$2/$IDX.tar.gz
    DST=$3/$IDX
    mkdir $DST
    tar -xvf $SRC -C $DST
done
