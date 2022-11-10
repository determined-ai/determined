pip install timm==0.3.1
# pycocotools 2.0.5, a dependency of efficientdet-pytorch,
# would not install without cython
pip install pycocotools==2.0.4
git clone https://github.com/rwightman/efficientdet-pytorch.git
cd efficientdet-pytorch
git checkout 611532db49fdd691f48f913bc433391a12014bd8
python setup.py install
cd ..
