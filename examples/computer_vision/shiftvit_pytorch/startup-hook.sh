git clone https://github.com/microsoft/SPACH.git
cd SPACH && git reset --hard f336b0d5258d71f53aefdd0a3b171e64b881296d
cd ..
mv SPACH/models/shiftvit.py .

python3 -m pip install einops timm
