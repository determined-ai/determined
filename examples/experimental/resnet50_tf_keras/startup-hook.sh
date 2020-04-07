URL='https://github.com/determined-ai/models.git'
FOLDER='/tmp/models'

if [ ! -d "$FOLDER" ] ; then
    git clone --depth 1 $URL $FOLDER
else
    cd "$FOLDER"
    git pull $URL
fi
export PYTHONPATH=${PYTHONPATH}:$FOLDER
