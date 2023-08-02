export PYTHONPATH=$PYTHONPATH:/gpt-neox

# Copy dataset from docker image to shared filesystem
USER=$(whoami)
mkdir /tmp/${USER}

cd /run/determined/workdir
cp gpt_neox_config/determined_cluster.yml /gpt-neox/configs
