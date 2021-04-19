# Product Week


## omnibar

- paste multiple cmds to execute.
- [~] sort out label and title usage. custom renderer? 
  - auto add space on confirmation?
- show output to the user
- use `help` cmd output as suggestions? 
- allow search through the fully resolve tree paths (help output)

- [x] reword and regroup sample tree.
- [x] generate command list for help doc.
- [x] get input from the user
- [x] hideomnibar through tree logic
- [x] improve sample tree
  - [x] add dev cmds


ops:
- context exp: kill, pause, archive, fork, cont, goto logs
  - show help
- kill experiment, task,
  - `experiment kill x`
  - `experiment open x`
  - `task kill x`
- [x] launch notebook, tensorboard
  - `notebook launch x`
  - `notebook kill x`
  - `tensorboard open trial/exp x` + comma to add more
  - `tens.. kill x`
- fork or continue
  - `exp/trial fork/contiue`
- [no] search experiment: nah we do it better with proper UI and filters
- [x] goto: 
  - exp
    - `goto experiment/trial x`: get the last one show by id
  - trial
  - task
  - single pages: exps, tasks, cluster, dashboard
- dev `! ` prefix
  - [x] set/reset server: `setServerAddress`, `resetServerAddress`, `showServerAddress`
  - reset browser pref: `resetBrowserPref`
  - [x] shareable ui: start/stop record, replay, export, import,
- logout
- show help



## shareable

- strip auth keys
- [p1] load from url with a sharable link pointing to a specific route in the ui.
  - [ ] import from url
  - intercept before routing? import and set mode to replay
- improve streaming endpoints capture
- [x] add to omnibar
  
