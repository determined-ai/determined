_base_ = "./fovea_r50_fpn_4x4_2x_coco.py"
model = dict(pretrained="torchvision://resnet101", backbone=dict(depth=101))
