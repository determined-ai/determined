pip install git+https://github.com/tensorflow/examples.git
# pip install tfds-nightly -- https://github.com/tensorflow/datasets/issues/1978 mentions that the nightly update should execute normally
python -m tensorflow_datasets.scripts.download_and_prepare --register_checksums=True --datasets='oxford_iiit_pet:3.*.*'
