pip install git+https://github.com/tensorflow/examples.git
pip install --upgrade absl-py
python -m tensorflow_datasets.scripts.download_and_prepare --register_checksums=True --datasets='oxford_iiit_pet:3.*.*'
