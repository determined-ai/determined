# Download the dataset from a reliable path that we control, rather than rely
# on public sources.  We do this in a startup-hook.sh hidden in the e2e fixture
# so the tutorial stays as easy-to-read as possible.

mkdir -p data
url="https://s3-us-west-2.amazonaws.com/determined-ai-test-data/torch_dataset_mnist.tgz"

# give it a few tries in case there are network failures
curl "$url" >data.tgz || curl "$url" >data.tgz || curl "$url" >data.tgz

tar -C data -xz <data.tgz

rm data.tgz
