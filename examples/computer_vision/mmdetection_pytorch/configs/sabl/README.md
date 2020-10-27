# Side-Aware Boundary Localization for More Precise Object Detection

## Introduction

We provide config files to reproduce the object detection results in the ECCV 2020 Spotlight paper for [Side-Aware Boundary Localization for More Precise Object Detection](https://arxiv.org/abs/1912.04260).

```
@inproceedings{Wang_2020_ECCV,
    title = {Side-Aware Boundary Localization for More Precise Object Detection},
    author = {Wang, Jiaqi and Zhang, Wenwei and Cao, Yuhang and Chen, Kai and Pang, Jiangmiao and Gong, Tao and Shi, Jianping, Loy, Chen Change and Lin, Dahua},
    booktitle = {ECCV},
    year = {2020}
}
```

## Results and Models

The results on COCO 2017 val is shown in the below table. (results on test-dev are usually slightly higher than val).
Single-scale testing (1333x800) is adopted in all results.


|       Method       | Backbone  | Lr schd | ms-train | box AP |                                                                                                                                                        Download                                                                                                                                                         |
| :----------------: | :-------: | :-----: | :------: | :----: | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: |
| SABL Faster R-CNN  | R-50-FPN  |   1x    |    N     |  39.9  |    [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_faster_rcnn_r50_fpn_1x_coco/sabl_faster_rcnn_r50_fpn_1x_coco-e867595b.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_faster_rcnn_r50_fpn_1x_coco/20200830_130324.log.json)    |
| SABL Faster R-CNN  | R-101-FPN |   1x    |    N     |  41.7  |  [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_faster_rcnn_r101_fpn_1x_coco/sabl_faster_rcnn_r101_fpn_1x_coco-f804c6c1.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_faster_rcnn_r101_fpn_1x_coco/20200830_183949.log.json)   |
| SABL Cascade R-CNN | R-50-FPN  |   1x    |    N     |  41.6  |  [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_cascade_rcnn_r50_fpn_1x_coco/sabl_cascade_rcnn_r50_fpn_1x_coco-e1748e5e.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_cascade_rcnn_r50_fpn_1x_coco/20200831_033726.log.json)   |
| SABL Cascade R-CNN | R-101-FPN |   1x    |    N     |  43.0  | [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_cascade_rcnn_r101_fpn_1x_coco/sabl_cascade_rcnn_r101_fpn_1x_coco-2b83e87c.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_cascade_rcnn_r101_fpn_1x_coco/20200831_141745.log.json) |

|     Method     | Backbone  |  GN   | Lr schd |  ms-train   | box AP |                                                                                                                                                                         Download                                                                                                                                                                         |
| :------------: | :-------: | :---: | :-----: | :---------: | :----: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: |
| SABL RetinaNet | R-50-FPN  |   N   |   1x    |      N      |  37.7  |                       [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r50_fpn_1x_coco/sabl_retinanet_r50_fpn_1x_coco-6c54fd4f.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r50_fpn_1x_coco/20200830_053451.log.json)                        |
| SABL RetinaNet | R-50-FPN  |   Y   |   1x    |      N      |  38.8  |                   [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r50_fpn_gn_1x_coco/sabl_retinanet_r50_fpn_gn_1x_coco-e16dfcf1.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r50_fpn_gn_1x_coco/20200831_141955.log.json)                   |
| SABL RetinaNet | R-101-FPN |   N   |   1x    |      N      |  39.7  |                      [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_1x_coco/sabl_retinanet_r101_fpn_1x_coco-42026904.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_1x_coco/20200831_034256.log.json)                      |
| SABL RetinaNet | R-101-FPN |   Y   |   1x    |      N      |  40.5  |                 [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_gn_1x_coco/sabl_retinanet_r101_fpn_gn_1x_coco-40a893e8.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_gn_1x_coco/20200830_201422.log.json)                  |
| SABL RetinaNet | R-101-FPN |   Y   |   2x    | Y (640~800) |  42.9  | [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_gn_2x_ms_640_800_coco/sabl_retinanet_r101_fpn_gn_2x_ms_640_800_coco-1e63382c.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_gn_2x_ms_640_800_coco/20200830_144807.log.json) |
| SABL RetinaNet | R-101-FPN |   Y   |   2x    | Y (480~960) |  43.6  | [model](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_gn_2x_ms_480_960_coco/sabl_retinanet_r101_fpn_gn_2x_ms_480_960_coco-5342f857.pth) &#124; [log](https://open-mmlab.s3.ap-northeast-2.amazonaws.com/mmdetection/v2.0/sabl/sabl_retinanet_r101_fpn_gn_2x_ms_480_960_coco/20200830_164537.log.json) |
