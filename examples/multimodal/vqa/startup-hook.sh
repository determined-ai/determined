apt-get update
apt-get install -y wget unzip xvfb
DEBIAN_FRONTEND=noninteractive apt-get install -y wkhtmltopdf
pip install h5py captum ipywidgets imgkit

wget https://s3.amazonaws.com/cvmlp/vqa/mscoco/vqa/Annotations_Train_mscoco.zip -O train_annotations.zip
wget https://s3.amazonaws.com/cvmlp/vqa/mscoco/vqa/Annotations_Val_mscoco.zip -O val_annotations.zip
wget https://s3.amazonaws.com/cvmlp/vqa/mscoco/vqa/Questions_Train_mscoco.zip -O train_questions.zip
wget https://s3.amazonaws.com/cvmlp/vqa/mscoco/vqa/Questions_Val_mscoco.zip -O val_questions.zip
wget https://s3.amazonaws.com/cvmlp/vqa/mscoco/vqa/Questions_Test_mscoco.zip -O test_questions.zip

mkdir vqa
unzip train_annotations.zip -d vqa/
unzip val_annotations.zip -d vqa/
unzip train_questions.zip -d vqa/
unzip val_questions.zip -d vqa/
unzip test_questions.zip -d vqa/

python preprocess_vocab.py

wget https://github.com/Cyanogenoid/pytorch-vqa/releases/download/v1.0/2017-08-04_00.55.19.pth -O 2017-08-04_00:55:19.pth
