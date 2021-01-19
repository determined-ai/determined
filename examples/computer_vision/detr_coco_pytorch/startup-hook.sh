apt-get update
apt-get install unzip

wget http://images.cocodataset.org/annotations/annotations_trainval2017.zip
unzip -o annotations_trainval2017.zip 
mv annotations/instances_train2017.json /tmp
mv annotations/instances_val2017.json /tmp

git clone https://github.com/facebookresearch/detr.git
cd detr && git reset --hard 4e1a9281bc5621dcd65f3438631de25e255c4269 && cd ..
pip install attrdict
pip install pycocotools
