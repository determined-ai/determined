#################################
 Configure Hyperparameter Ranges
#################################

The first step toward automatic hyperparameter tuning is to define the hyperparameter space, e.g.,
by :ref:`listing the decisions <topic-guides_hp-tuning-basics-example-hyperparameters>` that may
impact model performance. For each hyperparameter in the search space, the machine learning engineer
specifies a range of possible values in the experiment configuration:

.. code:: yaml

   hyperparameters:
     ...
     dropout_probability:
       type: double
       minval: 0.2
       maxval: 0.5
     ...

Determined supports the following searchable hyperparameter data types:

-  ``int``: an integer within a range
-  ``double``: a floating point number within a range
-  ``log``: a logarithmically scaled floating point number. Users specify a ``base``, and Determined
   searches the space of ``exponents`` within a range.
-  ``categorical``: a variable that can take on a value within a specified set of discrete values.
   The values themselves can be of any type.

The :ref:`experiment configuration reference <experiment-configuration_hyperparameters>` details
these data types and their associated options.

*********************************
 Configuring Searcher Parameters
*********************************

In addition to defining hyperparameter ranges, it's crucial to configure the searcher parameters
correctly. One of the most important parameters is ``max_length``.

max_length
==========

The ``max_length`` parameter specifies the maximum training length for any trial in the
hyperparameter search. It is a required parameter for the SearcherContext to function properly. When
configuring your experiment, ensure that you include ``max_length`` in the searcher section of your
configuration file:

.. code:: yaml

   searcher:
     name: adaptive_asha
     max_length:
       batches: 1000  # or any other appropriate value
     max_trials: 100
     metric: validation_loss
     smaller_is_better: true

The ``max_length`` value should be set based on your specific model and dataset. It should be long
enough for your model to converge but not so long that it wastes computational resources. The unit
(batches, epochs, records) should match the unit you use in your training code.

Remember that the SearcherContext now fabricates a single SearcherOperation based on this
``max_length`` value, so setting it correctly is crucial for the efficiency and effectiveness of
your hyperparameter search.
