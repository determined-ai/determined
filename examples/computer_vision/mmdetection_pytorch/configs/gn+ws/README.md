# Weight Standardization

## Introduction

```
@article{weightstandardization,
  author    = {Siyuan Qiao and Huiyu Wang and Chenxi Liu and Wei Shen and Alan Yuille},
  title     = {Weight Standardization},
  journal   = {arXiv preprint arXiv:1903.10520},
  year      = {2019},
}
```

## Results and Models

Faster R-CNN

| Backbone  | Style   | Normalization | Lr schd | Mem (GB) | Inf time (fps) | box AP | mask AP | Download |
|:---------:|:-------:|:-------------:|:-------:|:--------:|:--------------:|:------:|:-------:|:--------:|
| R-50-FPN  | pytorch | GN+WS         | 1x      | 5.9      | 11.7           | 39.7   | -       | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_r50_fpn_gn_ws-all_1x_coco/faster_rcnn_r50_fpn_gn_ws-all_1x_coco_20200130-613d9fe2.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_r50_fpn_gn_ws-all_1x_coco/faster_rcnn_r50_fpn_gn_ws-all_1x_coco_20200130_210936.log.json) |
| R-101-FPN | pytorch | GN+WS         | 1x      | 8.9      | 9.0            | 41.7   | -       | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_r101_fpn_gn_ws-all_1x_coco/faster_rcnn_r101_fpn_gn_ws-all_1x_coco_20200205-a93b0d75.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_r101_fpn_gn_ws-all_1x_coco/faster_rcnn_r101_fpn_gn_ws-all_1x_coco_20200205_232146.log.json) |
| X-50-32x4d-FPN | pytorch | GN+WS    | 1x      | 7.0      | 10.3           | 40.7   | -       | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_x50_32x4d_fpn_gn_ws-all_1x_coco/faster_rcnn_x50_32x4d_fpn_gn_ws-all_1x_coco_20200203-839c5d9d.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_x50_32x4d_fpn_gn_ws-all_1x_coco/faster_rcnn_x50_32x4d_fpn_gn_ws-all_1x_coco_20200203_220113.log.json) |
| X-101-32x4d-FPN | pytorch | GN+WS   | 1x      | 10.8     | 7.6            | 42.1   | -       | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_x101_32x4d_fpn_gn_ws-all_1x_coco/faster_rcnn_x101_32x4d_fpn_gn_ws-all_1x_coco_20200212-27da1bc2.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/faster_rcnn_x101_32x4d_fpn_gn_ws-all_1x_coco/faster_rcnn_x101_32x4d_fpn_gn_ws-all_1x_coco_20200212_195302.log.json) |

Mask R-CNN

| Backbone  | Style   | Normalization | Lr schd   | Mem (GB) | Inf time (fps) | box AP | mask AP | Download |
|:---------:|:-------:|:-------------:|:---------:|:--------:|:--------------:|:------:|:-------:|:--------:|
| R-50-FPN  | pytorch | GN+WS         | 2x        | 7.3      | 10.5       | 40.6        | 36.6    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r50_fpn_gn_ws-all_2x_coco/mask_rcnn_r50_fpn_gn_ws-all_2x_coco_20200226-16acb762.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r50_fpn_gn_ws-all_2x_coco/mask_rcnn_r50_fpn_gn_ws-all_2x_coco_20200226_062128.log.json) |
| R-101-FPN | pytorch | GN+WS         | 2x        | 10.3     | 8.6        | 42.0        | 37.7    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r101_fpn_gn_ws-all_2x_coco/mask_rcnn_r101_fpn_gn_ws-all_2x_coco_20200212-ea357cd9.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r101_fpn_gn_ws-all_2x_coco/mask_rcnn_r101_fpn_gn_ws-all_2x_coco_20200212_213627.log.json) |
| X-50-32x4d-FPN | pytorch | GN+WS    | 2x        | 8.4      | 9.3       | 41.1        | 37.0    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x50_32x4d_fpn_gn_ws-all_2x_coco/mask_rcnn_x50_32x4d_fpn_gn_ws-all_2x_coco_20200216-649fdb6f.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x50_32x4d_fpn_gn_ws-all_2x_coco/mask_rcnn_x50_32x4d_fpn_gn_ws-all_2x_coco_20200216_201500.log.json) |
| X-101-32x4d-FPN | pytorch | GN+WS   | 2x        | 12.2     | 7.1       | 42.1        | 37.9    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x101_32x4d_fpn_gn_ws-all_2x_coco/mask_rcnn_x101_32x4d_fpn_gn_ws-all_2x_coco_20200319-33fb95b5.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x101_32x4d_fpn_gn_ws-all_2x_coco/mask_rcnn_x101_32x4d_fpn_gn_ws-all_2x_coco_20200319_104101.log.json) |
| R-50-FPN  | pytorch | GN+WS         | 20-23-24e | 7.3      | -        | 41.1        | 37.1    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r50_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_r50_fpn_gn_ws-all_20_23_24e_coco_20200213-487d1283.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r50_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_r50_fpn_gn_ws-all_20_23_24e_coco_20200213_035123.log.json) |
| R-101-FPN | pytorch | GN+WS         | 20-23-24e | 10.3     | -        | 43.1        | 38.6    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r101_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_r101_fpn_gn_ws-all_20_23_24e_coco_20200213-57b5a50f.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_r101_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_r101_fpn_gn_ws-all_20_23_24e_coco_20200213_130142.log.json) |
| X-50-32x4d-FPN | pytorch | GN+WS    | 20-23-24e | 8.4      | -        | 42.1        | 38.0    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x50_32x4d_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_x50_32x4d_fpn_gn_ws-all_20_23_24e_coco_20200226-969bcb2c.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x50_32x4d_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_x50_32x4d_fpn_gn_ws-all_20_23_24e_coco_20200226_093732.log.json) |
| X-101-32x4d-FPN | pytorch | GN+WS   | 20-23-24e | 12.2     | -        | 42.7        | 38.5    | [model](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x101_32x4d_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_x101_32x4d_fpn_gn_ws-all_20_23_24e_coco_20200316-e6cd35ef.pth) &#124; [log](http://download.openmmlab.com/mmdetection/v2.0/gn%2Bws/mask_rcnn_x101_32x4d_fpn_gn_ws-all_20_23_24e_coco/mask_rcnn_x101_32x4d_fpn_gn_ws-all_20_23_24e_coco_20200316_013741.log.json) |

Note:

- GN+WS requires about 5% more memory than GN, and it is only 5% slower than GN.
- In the paper, a 20-23-24e lr schedule is used instead of 2x.
- The X-50-GN and X-101-GN pretrained models are also shared by the authors.
