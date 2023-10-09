.. _hyperparameter-tuning:

.. _topic-guides_hp-tuning-basics:

#######################
 Hyperparameter Tuning
#######################

Hyperparameter tuning is the common machine learning process of selecting the data, features, model
architecture, and learning algorithm to yield an effective model. Hyperparameter tuning is a
:ref:`challenging problem <topic-guides_hp-tuning-basics-difficulty>` given the potentially large
number of hyperparameters to consider.

********************************
 Why Do Hyperparameters Matter?
********************************

.. _topic-guides_hp-tuning-basics-example-hyperparameters:

During the model development lifecycle, a machine learning engineer makes a wide range of decisions
impacting model performance. For example, a computer vision model requires decisions on sample
features, model architecture, and training algorithm parameters, e.g.:

-  Should we consider features aside from the raw images in the training set?

   -  Would synthetic data augmentation techniques like image rotation or horizontal flipping yield
      a better model?
   -  Should we populate additional features via advanced image processing techniques such as shape
      edge extraction?

-  What model architecture works best?

   -  How many layers?
   -  What kind of layers (e.g., dense, dropout, pooling)?
   -  How should we parameterize each layer (e.g., size, activation function)?

-  What learning algorithm hyperparameters should we use?

   -  What gradient descent batch size should we use?
   -  What optimizer should we use, and how should we parameterize it (e.g., learning rate)?

A machine learning engineer can manually guess and test hyperparameters, or they might narrow the
search space by using a pretrained model. However, even if the engineer achieves seemingly good
model performance, they're left wondering how much better they might do with additional tuning.

***********************
 Hyperparameter Tuning
***********************

.. _topic-guides_hp-tuning-basics-difficulty:

Hyperparameter tuning is a crucial phase in the model development lifecycle. However, tuning deep
learning models is difficult because:

-  A deep learning model's objective (e.g., validation loss) as a function of the hyperparameters is
   non-continuous and noisy, so we can't apply analytical or continuous optimization techniques to
   calculate the validation objective given a set of hyperparameters. Thus, hyperparameter tuning is
   a black box optimization problem in that we must train a model under a set of hyperparameters in
   order to evaluate the objective.

-  Hyperparameter tuning suffers from the curse of dimensionality, as the number of possible
   hyperparameter configurations is exponential in the number of hyperparameters. For instance, even
   if a model has just ten categorical hyperparameters with five values per hyperparameter, and each
   hyperparameter configuration takes one minute to train on average, it would take 5^10 minutes, or
   nearly 20 years, to evaluate all possible hyperparameter configurations.

-  Deep learning model training is computationally expensive. It's not uncommon for a model to
   require hours or days to train, even on expensive hardware.

Fortunately, there are automatic hyperparameter tuning techniques that the machine learning engineer
can leverage to find an effective model. Determined provides support for hyperparameter search as a
first-class workflow that is tightly integrated with Determined's job scheduler, which allows for
efficient execution of state-of-the-art early-stopping based approaches as well as seamless
parallelization of these methods.

An intuitive interface is provided to use hyperparameter searching, and is described in the
following sections.

.. toctree::
   :maxdepth: 1
   :hidden:

   configure-hp-ranges
   hp-constraints-det
   instrument-model-code
   handle-trial-errors
   Search Methods <search-methods/_index>
