pip install git+https://github.com/tensorflow/examples.git
pip install -q -U tfds-nightly
python -m tensorflow_datasets.scripts.download_and_prepare --register_checksums=True --datasets='oxford_iiit_pet:3.*.*' --data_dir='/tmp/data'

