:orphan:

**New Features**

-  WebUI: Turn on new run-centric search view

   -  This replaces the existing experiment search and multi-trial experiment views with views that
      allow comparison between arbitrary trials.

   -  We recieved user feedback indicating that users of Determined run primarily single-trial
      experiments, and the preexisting views made comparing model performance between different
      trials difficult.

   -  We are renaming experiments to searches and trials to runs to make the difference between the
      two easier to understand.

   -  The experiment list is now the run list, which shows all trials from experiments in the
      project. It should function similar to the previous new experiment list.

   -  Multi-trial experiments can be viewed in the new searches view, which allows for sorting,
      filtering and navigation of the multi-trial experiments in the project.

   -  When viewing a multi-trial experiment, we now show a list of trials in the experiment to allow
      for sorting, filtering and arbitrary comparison.
