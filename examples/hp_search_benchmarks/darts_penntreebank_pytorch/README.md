# NAS with state-of-the-art HP Search
Determined's adaptive search method implements [ASHA](https://arxiv.org/pdf/1810.05934.pdf), 
a state-of-the-art method for HP search suitable for large-scale machine learning.  In this example, 
we use Determined's adaptive HP search to search for CNN architectures from [a common search space 
used for neural architecture search (NAS)](https://arxiv.org/abs/1806.09055).  In particular, we replicate 
the NAS RNN search benchmark from the ASHA paper (Figure 4), which is [a strong baseline 
for NAS](https://arxiv.org/abs/1902.07638).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model. 
* **optimizer.py**: The optimizer code for the model. 
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data
This example searches for architectures over the [Penn Treebank 
dataset](https://pytorchnlp.readthedocs.io/en/latest/_modules/torchnlp/datasets/penn_treebank.html),
but is easily adaptable to other datasets.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
adaptive.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `adaptive.yaml`.

### Expected Performance
With 16 V100 GPUs, the best architecture after 10 hours should achieve a test perplexity of around 65.  
For a fair comparison to the NAS results for this search space, you will have to train the best architecture 
following the extended training routine outlined in [this paper](https://arxiv.org/abs/1806.09055).
