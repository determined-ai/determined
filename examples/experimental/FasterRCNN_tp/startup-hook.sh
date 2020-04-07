git clone --depth 1 https://github.com/cocodataset/cocoapi
pip install -e cocoapi/PythonAPI
pip install opencv-python-headless==4.1.0.25
curl -o /root/ImageNet-R50-AlignPadding.npz https://storage.googleapis.com/determined-ai-coco-dataset/ImageNet-R50-AlignPadding.npz
