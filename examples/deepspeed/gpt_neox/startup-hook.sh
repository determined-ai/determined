export PYTHONPATH=$PYTHONPATH:/gpt-neox

# Use data in docker image and copy over needed configs.
USER=$(whoami)
mkdir /tmp/${USER}
cd /run/determined/workdir
cp gpt_neox_config/* /gpt-neox/configs
