#!/bin/bash

set -o errexit
set -o xtrace

cd /tmp

if [ -d rcnn-data ]; then
    exit
fi
mkdir rcnn-data
cd rcnn-data

curl -O http://images.cocodataset.org/zips/train2014.zip \
     -O http://images.cocodataset.org/zips/val2014.zip \
     -O http://images.cocodataset.org/annotations/annotations_trainval2014.zip \
     -O http://models.tensorpack.com/FasterRCNN/ImageNet-R50-AlignPadding.npz \
     -o instances_minival2014.json.zip https://dl.dropboxusercontent.com/s/o43o90bna78omob/instances_minival2014.json.zip?dl=1 \
     -o instances_valminusminival2014.json.zip https://dl.dropboxusercontent.com/s/s3tw5zcg7395368/instances_valminusminival2014.json.zip?dl=1

mkdir -p COCO/DIR
unzip -qd COCO/DIR train2014.zip
unzip -qd COCO/DIR val2014.zip
unzip -qd COCO/DIR annotations_trainval2014.zip
unzip -qd COCO/DIR/annotations instances_minival2014.json.zip
unzip -qd COCO/DIR/annotations instances_valminusminival2014.json.zip
