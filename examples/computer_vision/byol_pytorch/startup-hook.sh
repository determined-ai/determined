# Copy LARS implementation from upstream repo.
git clone https://github.com/untitled-ai/self_supervised.git
(cd self_supervised && git checkout 6d14ca0402ecc13feda9b3a9fdc056fd1ac24473)
cp self_supervised/lars.py ./
python3 -m pip install attrdict byol-pytorch filelock
