.. _model-hub-transformers-examples:

##########
 Examples
##########

`Transformers examples <https://github.com/huggingface/transformers/tree/master/examples>`_ are the
starting point for many users of the `transformers <https://github.com/huggingface/transformers>`_
library. Hence, they are the core feature of **model-hub's** support for transformers library. In
fact, each `transformer example corresponding to a core transformer task
<https://huggingface.co/transformers/examples.html#the-big-table-of-tasks>`_ has an associated task
in **model-hub** that is guaranteed to work with Determined and verified for correctness. See the
table below for a summary of the **model-hub** transformers examples:

.. list-table::
   :header-rows: 1

   -  -  Task
      -  Dataset
      -  Filename

   -  -  language-modeling
      -  WikiText-2
      -  :download:`language-modeling.tgz </examples/language-modeling.tgz>`

   -  -  multiple-choice
      -  SWAG
      -  :download:`multiple-choice.tgz </examples/multiple-choice.tgz>`

   -  -  question-answering
      -  SQuAD and SQuAD version 2
      -  :download:`question-answering.tgz </examples/question-answering.tgz>`

   -  -  text-classification
      -  GLUE and XNLI
      -  :download:`text-classification.tgz </examples/text-classification.tgz>`

   -  -  token-classification
      -  CoNLL-2003
      -  :download:`token-classification.tgz </examples/token-classification.tgz>`

   -  -  summarization
      -  CNN/DailyMail and XSum
      -  Coming Soon

   -  -  translation
      -  WMT-16
      -  Coming Soon

Each of the Determined trials above subclasses the ``BaseTransformerTrial`` and makes use of helper
functions in ``model_hub.huggingface`` to drastically reduce the code needed to run the example. See
:ref:`the api <model-hub-transformers-api>` for more details.
