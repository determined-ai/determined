_base_ = "./mask_rcnn_r50_fpn_1x_coco.py"
model = dict(pretrained="torchvision://resnet101", backbone=dict(depth=101))
