## TODO

- [x] update jobs in each scheduler individually
- [x] remove Orderedallocation interfaces
- [x] probe rp for job list from jobs
- [ ] pr!
- [ ] p2: actor alias
- [x] add from proto translation
- [x] add name and user information for jobs
- [x] low cost single job summary lookup
- [x] scheduling state
- [ ] cache jobinfo in jobs actor?
- [k] remove isPrememptible from rmjobiinfo, read from config
- [x] read prio, weight, qvalue from job actor
  - [ ] group usage: noOp? we probably still want to store on job actor and propagate down the same way group is set. 


chat:
- async update pattern of q info
- multiple queues per scheduler (leave out of this pr)
- jobinfo struct

## actor system
### tip of master:
root
  - experiments
    - exp 1 actor (aka a job)
    - exp 2 atcor
  - notebooks
    - cmd1, ..
  - tensorboards, ..

### this pr
root
  - experiments
    - exp 1 actor (aka a job)
    - exp 2 atcor
  - notebooks
    - cmd1, ..
  - jobs: pass messages for jobs directly to children of experiments, command, notebooks eg [exp1, exp2, cmd1]. I considered registering exp1, cmd1 directly as chilldren but not sure if our actor system would like that
    - no official children

### potential future
root
  - jobs: pass messages for jobs directly to children of experiments, command, notebooks eg [exp1, exp2, cmd1]
    - experiments
      - exp 1 actor (aka a job)
      - exp 2 atcor
    - notebooks
      - cmd1, ..

