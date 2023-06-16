#!/bin/sh

taskid="1.49d9b3e3-bbc2-46a3-8ef1-b2193e75462d"

for i in `seq 100000` ; do
printf "insert into trials (id, experiment_id, state, start_time, hparams, task_id) values (%d, 1, 'ERROR', '2023-07-25 16:44:21.610081+00', '{}', '$taskid');\n" $((i*4/3 + 10))
done
