# Very important to pin sentencepiece as the newer version cuases segementation faults (as of Oct 2020)
pip install sentencepiece==0.1.91
pip install transformers==3.1.0
pip install -e git+git://github.com/LiyuanLucasLiu/RAdam.git@baf4f65445c00d686d4098841b3ca1f62a886326#egg=radam
