apt-get update
apt-get install unzip

# Download COCO 2017 annotations
wget http://images.cocodataset.org/annotations/annotations_trainval2017.zip
unzip -o annotations_trainval2017.zip 
mv annotations/instances_train2017.json /tmp
mv annotations/instances_val2017.json /tmp

# Clone Deformable-DETR library from source.  
# Since it is not an installable pacakge, we will have to add this to system path to import functions from it.
git clone https://github.com/fundamentalvision/Deformable-DETR ddetr
cd ddetr && git reset --hard 11169a60c33333af00a4849f1808023eba96a931 

pip install tqdm attrdict pycocotools cython scipy

# Build custom cuda ops
cd models/ops 
sh ./make.sh
cd ../../..

# Download pretrained model using link from https://github.com/fundamentalvision/Deformable-DETR
pip install gdown
gdown https://drive.google.com/uc?id=1nDWZWHuRwtwGden77NLM9JoWe-YisJnA -O model.ckpt  
