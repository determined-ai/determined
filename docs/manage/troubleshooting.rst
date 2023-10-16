.. _troubleshooting:

#################
 Troubleshooting
#################

****************
 Error messages
****************

.. code::

   docker: Error response from daemon: OCI runtime create failed: container_linux.go:345: starting container process caused "process_linux.go:424: container init caused \"process_linux.go:407: running prestart hook 1 caused \\\"error running hook: exit status 1, stdout: , stderr: exec command: [/usr/bin/nvidia-container-cli --load-kmods configure --ldconfig=@/sbin/ldconfig --device=all --compute --utility --require=cuda>=10.0 brand=tesla,driver>=384,driver<385 brand=tesla,driver>=410,driver<411 --pid=35777 /var/lib/docker/devicemapper/mnt/7b5b6d59cd4fe9307b7523f1cc9ce3bc37438cc793ff4a5a18a0c0824ec03982/rootfs]\\\\nnvidia-container-cli: requirement error: unsatisfied condition: brand = tesla\\\\n\\\"\"": unknown.

This error message indicates that the GPU hardware and/or NVIDIA drivers installed on the agent are
not compatible with CUDA 10, and you are trying to run a Docker image that depends on CUDA 10.

To resolve this issue, run the following commands. If the first succeeds and the second fails, you
should be able to use Determined as long as you use Docker images based on CUDA 9.

.. code::

   docker run --gpus all --rm nvidia/cuda:9.0-runtime nvidia-smi
   docker run --gpus all --rm nvidia/cuda:10.0-runtime nvidia-smi

***********************************
 Debug Database Migration Failures
***********************************

.. code::

   Dirty database version <a long number>. Fix and force version.

If you see the above error message, a database migration was likely interrupted while running and
the database is now in a dirty state.

Make sure you back up the database and temporarily shut down the master before proceeding further.

To fix this error message, locate the up migration with a suffix of ``.up.sql`` and a prefix
matching the long number in the error message in `this directory
<https://github.com/determined-ai/determined/tree/master/master/static/migrations>_` and carefully
run the SQL within the file manually against the database used by Determined. For convenience, all
the information needed to connect except the password can be found with:

.. code::

   det master config | jq .db

If this proceeds successfully, then mark the migration as successful by running the following SQL:

.. code::

   UPDATE schema_migrations SET dirty = false;

And restart the master. Otherwise, please seek assistance in the community `Slack
<https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew>`__.

.. _validate-nvidia-container-toolkit:

***********************************
 Validate NVIDIA Container Toolkit
***********************************

To verify that a Determined agent instance can run containers that use GPUs, run:

.. code::

   docker run --gpus all --rm debian:10-slim nvidia-smi

You should see output that describes the GPUs available on the agent instance, such as:

.. code::

   +-----------------------------------------------------------------------------+
   | NVIDIA-SMI 418.39       Driver Version: 418.39       CUDA Version: 10.1     |
   |-------------------------------+----------------------+----------------------+
   | GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
   | Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
   |===============================+======================+======================|
   |   0  GeForce GTX 108...  Off  | 00000000:05:00.0 Off |                  N/A |
   | 56%   84C    P2   177W / 250W |  10729MiB / 11176MiB |     76%      Default |
   +-------------------------------+----------------------+----------------------+
   |   1  GeForce GTX 108...  Off  | 00000000:06:00.0 Off |                  N/A |
   | 28%   62C    P0    56W / 250W |      0MiB / 11178MiB |      0%      Default |
   +-------------------------------+----------------------+----------------------+
   |   2  GeForce GTX 108...  Off  | 00000000:09:00.0 Off |                  N/A |
   | 31%   64C    P0    57W / 250W |      0MiB / 11178MiB |      0%      Default |
   +-------------------------------+----------------------+----------------------+
   |   3  TITAN Xp            Off  | 00000000:0A:00.0 Off |                  N/A |
   | 20%   36C    P0    57W / 250W |      0MiB / 12196MiB |      6%      Default |
   +-------------------------------+----------------------+----------------------+

   +-----------------------------------------------------------------------------+
   | Processes:                                                       GPU Memory |
   |  GPU       PID   Type   Process name                             Usage      |
   |=============================================================================|
   |    0      4638      C   python3                                    10719MiB |
   +-----------------------------------------------------------------------------+
