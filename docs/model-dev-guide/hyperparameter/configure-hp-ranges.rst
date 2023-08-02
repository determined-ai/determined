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
