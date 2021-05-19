# Tensorflow Multi-class Text Classification Example

This example demonstrates how to integrate and train a Tensorflow model on Determined's API. 
We will implement a multiclass classifier that will be trained to perform sentiment 
analysis on a dataset of Amazon product reviews, classifying reviews into 5 categories (0-5 stars). 
It is adapted from Tensorflow's
[word embedding](https://www.tensorflow.org/tutorials/text/word_embeddings) tutorial.

## Files
* `model_def.py`: The core code for training. This includes building and compiling the model.
* `data.py`: The data loading and preparation code for the model.

###Configuration Files
* `const.yaml`: Train the model on a single GPU with constant hyperparameter values.
* `distributed.yaml`: Same as const.yaml, but trains the model with multiple GPUs.
* `adaptive.yaml`: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

##Data
This example uses [UCSD's Amazon review datasets](http://deepyeti.ucsd.edu/jianmo/amazon/) but can be easily adapted to 
use any other categorical text-based dataset.

##Running the Experiment
If you have not yet installed Determined, installation instructions can be found under docs/install-admin.html or at 
https://docs.determined.ai/latest/index.html

Run the following command:

`det -m <master host:port> experiment create -f const.yaml .`

The other configurations can be run by specifying the appropriate configuration file in place of adaptive.yaml.

##Sample Results
Example of running a trial with default configurations in `const.yaml` and expected convergence.

![Expected convergence](results.png)
