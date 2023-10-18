apt-get update
apt-get install unzip

wget http://images.cocodataset.org/annotations/annotations_trainval2017.zip
unzip -o annotations_trainval2017.zip
mv annotations/instances_train2017.json /tmp
mv annotations/instances_val2017.json /tmp

git clone https://github.com/facebookresearch/detr.git
cd detr && git reset --hard 4e1a9281bc5621dcd65f3438631de25e255c4269
# Need to fix a bug in the original code that fails to handle torchvision version 0.10 correctly.
sed -i 's/float(torchvision\.__version__\[:3\]) < 0.7/int(torchvision\.__version__.split("\.")\[1\]) < 7/g' util/misc.py
cd ..

pip install attrdict
pip install pycocotools
