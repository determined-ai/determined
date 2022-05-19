.. _model-registry:

################
 Model Registry
################

The Model Registry is a way to group together conceptually related checkpoints (including ones
across different experiments), storing metadata and longform notes about a model, and retrieving the
latest version of a model for use or futher development. The Model Registry can be accessed through
the WebUI, Python API, REST API, or CLI, though the WebUI has some features that the others are
missing.

*******
 WebUI
*******

The Model Registry is a top-level option in the navigation bar. This will take you to a page listing
all of the models that currently exist in the registry, and allow you to create new models. You can
select any of the existing models to go to the Model Details page, where you can view and edit
detailed information about the model. There will also be a list of every version associated with the
selected model, and you can go to the Version Details page to view and edit that version's
information.

For more information about how to use the model registry, see `Organizing Models in the Model
Registry <../post-training/model-registry.html>`_
