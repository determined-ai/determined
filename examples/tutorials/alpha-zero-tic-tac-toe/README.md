# Alpha Zero Tic-Tac-Toe

See https://github.com/suragnair/alpha-zero-general for original source.

To verify results of an experiment download a checkpoint and run `pit.py`. For example:

```
# launch an experiment
det experiment create 1_const.yaml .
# wait for experiment to complete
# download a checkpoint
det checkpoint download some-checkpoint-uuid
# relocate checkpoint contents
mv checkpoints/*/checkpoints/* checkpoints/
# test againt perfect player
python pit.py
```
