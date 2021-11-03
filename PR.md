# tip of master:
root
  - experiments
    - exp 1 actor (aka a job)
    - exp 2 atcor
  - notebooks
    - cmd1, ..
  - tensorboards, ..

# this pr
root
  - experiments
    - exp 1 actor (aka a job)
    - exp 2 atcor
  - notebooks
    - cmd1, ..
  - jobs: pass messages for jobs directly to children of experiments, command, notebooks eg [exp1, exp2, cmd1]. I considered registering exp1, cmd1 directly as chilldren but not sure if our actor system would like that
    - no official children

# potential future
root
  - jobs: pass messages for jobs directly to children of experiments, command, notebooks eg [exp1, exp2, cmd1]
    - experiments
      - exp 1 actor (aka a job)
      - exp 2 atcor
    - notebooks
      - cmd1, ..
