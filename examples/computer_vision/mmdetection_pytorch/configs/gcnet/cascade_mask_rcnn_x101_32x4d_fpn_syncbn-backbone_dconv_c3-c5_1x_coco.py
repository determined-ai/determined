_base_ = "../dcn/cascade_mask_rcnn_r50_fpn_dconv_c3-c5_1x_coco.py"
model = dict(
    backbone=dict(norm_cfg=dict(type="SyncBN", requires_grad=True), norm_eval=False)
)
