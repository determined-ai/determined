_base_ = "./faster_rcnn_hrnetv2p_w40_1x_coco.py"
# learning policy
lr_config = dict(step=[16, 22])
total_epochs = 24
