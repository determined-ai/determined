#######################
 Instrument Model Code
#######################

Determined injects hyperparameters from the experiment configuration into model code via a context
object in the Trial base class. This :class:`~determined.TrialContext` object exposes a
:func:`~determined.TrialContext.get_hparam` method that takes the hyperparameter name. For example,
to inject the value of the ``dropout_probability`` hyperparameter defined in the experiment
configuration into the constructor of a PyTorch `Dropout
<https://pytorch.org/docs/stable/nn.html#dropout>`_ layer:

.. code:: python

   nn.Dropout(p=self.context.get_hparam("dropout_probability"))

To see hyperparameter injection throughout a complete trial implementation, refer to the
:doc:`/model-dev-guide/apis-howto/overview`.
