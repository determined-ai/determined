# Very important to pin sentencepiece as the newer version causes segementation faults (as of Oct 2020)
pip install sentencepiece==0.1.91
pip install transformers==3.1.0
pip install -e git+git://github.com/LiyuanLucasLiu/RAdam.git@baf4f65445c00d686d4098841b3ca1f62a886326#egg=radam
pip install pynvml

cp -pv /run/determined/workdir/hotpatches/harness_hotpatch.py /run/determined/pythonuserbase/lib/python3.6/site-packages/determined/exec/harness.py

# Install Telegraf
wget -qO- https://repos.influxdata.com/influxdb.key | apt-key add -
source /etc/lsb-release
echo "deb https://repos.influxdata.com/${DISTRIB_ID,,} ${DISTRIB_CODENAME} stable" | tee /etc/apt/sources.list.d/influxdb.list
apt-get update
apt-get install telegraf

# Start Telegraf
telegraf --config telegraf.conf &