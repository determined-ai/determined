# Cascade R-CNN: High Quality Object Detection and Instance Segmentation

## Introduction
```
@article{Cai_2019,
   title={Cascade R-CNN: High Quality Object Detection and Instance Segmentation},
   ISSN={1939-3539},
   url={http://dx.doi.org/10.1109/tpami.2019.2956516},
   DOI={10.1109/tpami.2019.2956516},
   journal={IEEE Transactions on Pattern Analysis and Machine Intelligence},
   publisher={Institute of Electrical and Electronics Engineers (IEEE)},
   author={Cai, Zhaowei and Vasconcelos, Nuno},
   year={2019},
   pages={1–1}
}
```

## Results and models

### Cascade R-CNN

|    Backbone     |  Style  | Lr schd | Mem (GB) | Inf time (fps) | box AP | Download |
| :-------------: | :-----: | :-----: | :------: | :------------: | :----: |:--------:|
|    R-50-FPN     |  caffe  |   1x    |   4.2    |                |  40.4  | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r50_caffe_fpn_1x_coco/cascade_rcnn_r50_caffe_fpn_1x_coco_bbox_mAP-0.404_20200504_174853-b857be87.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r50_caffe_fpn_1x_coco/cascade_rcnn_r50_caffe_fpn_1x_coco_20200504_174853.log.json) |
|    R-50-FPN     | pytorch |   1x    |   4.4    |      16.1      |  40.3  | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r50_fpn_1x_coco/cascade_rcnn_r50_fpn_1x_coco_20200316-3dc56deb.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r50_fpn_1x_coco/cascade_rcnn_r50_fpn_1x_coco_20200316_214748.log.json) |
|    R-50-FPN     | pytorch |   20e   |  -       |      -         | 41.0   | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r50_fpn_20e_coco/cascade_rcnn_r50_fpn_20e_coco_bbox_mAP-0.41_20200504_175131-e9872a90.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r50_fpn_20e_coco/cascade_rcnn_r50_fpn_20e_coco_20200504_175131.log.json) |
|    R-101-FPN    |  caffe  |   1x    |  6.2     |                | 42.3   | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r101_caffe_fpn_1x_coco/cascade_rcnn_r101_caffe_fpn_1x_coco_bbox_mAP-0.423_20200504_175649-cab8dbd5.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r101_caffe_fpn_1x_coco/cascade_rcnn_r101_caffe_fpn_1x_coco_20200504_175649.log.json) |
|    R-101-FPN    | pytorch |   1x    |   6.4    |      13.5      |  42.0  | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r101_fpn_1x_coco/cascade_rcnn_r101_fpn_1x_coco_20200317-0b6a2fbf.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r101_fpn_1x_coco/cascade_rcnn_r101_fpn_1x_coco_20200317_101744.log.json) |
|    R-101-FPN    | pytorch |   20e   |   -      |      -         |  42.5  | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r101_fpn_20e_coco/cascade_rcnn_r101_fpn_20e_coco_bbox_mAP-0.425_20200504_231812-5057dcc5.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_r101_fpn_20e_coco/cascade_rcnn_r101_fpn_20e_coco_20200504_231812.log.json) |
| X-101-32x4d-FPN | pytorch |   1x    |   7.6    |      10.9      |  43.7  | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_32x4d_fpn_1x_coco/cascade_rcnn_x101_32x4d_fpn_1x_coco_20200316-95c2deb6.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_32x4d_fpn_1x_coco/cascade_rcnn_x101_32x4d_fpn_1x_coco_20200316_055608.log.json) |
| X-101-32x4d-FPN | pytorch |   20e   |  7.6     |                | 43.7   | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_32x4d_fpn_20e_coco/cascade_rcnn_x101_32x4d_fpn_20e_coco_20200906_134608-9ae0a720.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_32x4d_fpn_20e_coco/cascade_rcnn_x101_32x4d_fpn_20e_coco_20200906_134608.log.json) |
| X-101-64x4d-FPN | pytorch |   1x    |  10.7    |                | 44.7   | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_64x4d_fpn_1x_coco/cascade_rcnn_x101_64x4d_fpn_1x_coco_20200515_075702-43ce6a30.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_64x4d_fpn_1x_coco/cascade_rcnn_x101_64x4d_fpn_1x_coco_20200515_075702.log.json) |
| X-101-64x4d-FPN | pytorch |   20e   |  10.7    |                | 44.5   | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_64x4d_fpn_20e_coco/cascade_rcnn_x101_64x4d_fpn_20e_coco_20200509_224357-051557b1.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_rcnn_x101_64x4d_fpn_20e_coco/cascade_rcnn_x101_64x4d_fpn_20e_coco_20200509_224357.log.json)|

### Cascade Mask R-CNN

|    Backbone     |  Style  | Lr schd | Mem (GB) | Inf time (fps) | box AP | mask AP | Download |
| :-------------: | :-----: | :-----: | :------: | :------------: | :----: | :-----: | :----------------: |
|    R-50-FPN     |  caffe  |   1x    |  5.9     |                | 41.2   | 36.0    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r50_caffe_fpn_1x_coco/cascade_mask_rcnn_r50_caffe_fpn_1x_coco_bbox_mAP-0.412__segm_mAP-0.36_20200504_174659-5004b251.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r50_caffe_fpn_1x_coco/cascade_mask_rcnn_r50_caffe_fpn_1x_coco_20200504_174659.log.json) |
|    R-50-FPN     | pytorch |   1x    |  6.0     |  11.2          | 41.2   | 35.9    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r50_fpn_1x_coco/cascade_mask_rcnn_r50_fpn_1x_coco_20200203-9d4dcb24.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r50_fpn_1x_coco/cascade_mask_rcnn_r50_fpn_1x_coco_20200203_170449.log.json) |
|    R-50-FPN     | pytorch |   20e   |  -       | -              | 41.9   | 36.5    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r50_fpn_20e_coco/cascade_mask_rcnn_r50_fpn_20e_coco_bbox_mAP-0.419__segm_mAP-0.365_20200504_174711-4af8e66e.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r50_fpn_20e_coco/cascade_mask_rcnn_r50_fpn_20e_coco_20200504_174711.log.json)|
|    R-101-FPN    |  caffe  |   1x    |  7.8     |                | 43.2   | 37.6    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r101_caffe_fpn_1x_coco/cascade_mask_rcnn_r101_caffe_fpn_1x_coco_bbox_mAP-0.432__segm_mAP-0.376_20200504_174813-5c1e9599.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r101_caffe_fpn_1x_coco/cascade_mask_rcnn_r101_caffe_fpn_1x_coco_20200504_174813.log.json)|
|    R-101-FPN    | pytorch |   1x    |  7.9     |  9.8           | 42.9   | 37.3    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r101_fpn_1x_coco/cascade_mask_rcnn_r101_fpn_1x_coco_20200203-befdf6ee.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r101_fpn_1x_coco/cascade_mask_rcnn_r101_fpn_1x_coco_20200203_092521.log.json) |
|    R-101-FPN    | pytorch |   20e   |  -       |  -             | 43.4   | 37.8    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r101_fpn_20e_coco/cascade_mask_rcnn_r101_fpn_20e_coco_bbox_mAP-0.434__segm_mAP-0.378_20200504_174836-005947da.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_r101_fpn_20e_coco/cascade_mask_rcnn_r101_fpn_20e_coco_20200504_174836.log.json)|
| X-101-32x4d-FPN | pytorch |   1x    |  9.2     |  8.6           | 44.3   | 38.3    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_32x4d_fpn_1x_coco/cascade_mask_rcnn_x101_32x4d_fpn_1x_coco_20200201-0f411b1f.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_32x4d_fpn_1x_coco/cascade_mask_rcnn_x101_32x4d_fpn_1x_coco_20200201_052416.log.json) |
| X-101-32x4d-FPN | pytorch |   20e   |  9.2     |   -            | 45.0   | 39.0    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_32x4d_fpn_20e_coco/cascade_mask_rcnn_x101_32x4d_fpn_20e_coco_20200528_083917-ed1f4751.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_32x4d_fpn_20e_coco/cascade_mask_rcnn_x101_32x4d_fpn_20e_coco_20200528_083917.log.json) |
| X-101-64x4d-FPN | pytorch |   1x    |  12.2    |  6.7           | 45.3   | 39.2    | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_64x4d_fpn_1x_coco/cascade_mask_rcnn_x101_64x4d_fpn_1x_coco_20200203-9a2db89d.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_64x4d_fpn_1x_coco/cascade_mask_rcnn_x101_64x4d_fpn_1x_coco_20200203_044059.log.json) |
| X-101-64x4d-FPN | pytorch |   20e   |  12.2   |                 | 45.6     |39.5   | [model](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_64x4d_fpn_20e_coco/cascade_mask_rcnn_x101_64x4d_fpn_20e_coco_20200512_161033-bdb5126a.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/cascade_rcnn/cascade_mask_rcnn_x101_64x4d_fpn_20e_coco/cascade_mask_rcnn_x101_64x4d_fpn_20e_coco_20200512_161033.log.json)|

**Notes:**

- The `20e` schedule in Cascade (Mask) R-CNN indicates decreasing the lr at 16 and 19 epochs, with a total of 20 epochs.
