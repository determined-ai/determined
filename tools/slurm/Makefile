.PHONY: usage
usage:
	@echo "NAME"
	@echo "    make slurmcluster"
	@echo " "
	@echo "SYNOPSIS"
	@echo "    make slurmcluster [FLAGS=\"options\"] [TF_LOCK=\"value\"]"
	@echo " "
	@echo "DESCRIPTION"
	@echo "    Launches a compute instance with Slurm, Singularity (Apptainer), the Cray"
	@echo "    Launcher component, and many other dependencies pre-installed. Then, SSH tunnels"
	@echo "    are opened so that localhost:8081 on your machine points at port 8081 on"
	@echo "    the compute instance and port 8080 on the compute instance points at"
	@echo "    localhost:8080 on your machine. Lastly, devcluster is started with the Slurm"
	@echo "    RM pointed at the remote instance, and local development with devcluster works"
	@echo "    as always."
	@echo " "
	@echo "OPTIONS"
	@echo "    FLAGS"
	@echo "        Pass these options to the FLAGS variable. The FLAGS variable"
	@echo "        is passed into the tools/slurmcluster.sh script."
	@echo "        -A  Invokes a slurmcluster that uses agents instead of the launcher."
	@echo "        -c  Invokes a slurmcluster using the specified container run type."
	@echo "            Options are 'enroot', 'podman', or 'singularity'. Default is 'singularity'."
	@echo "        -w  Invokes a slurmcluster using the specified workload manager."
	@echo "            Options are 'slurm' or 'pbs'. Default is 'slurm'."
	@echo "        -t  Sets a time limit in seconds for the slurmcluster devbox to run"
	@echo "            before automatically deleting the VM. If unspecified, the default time is two"
	@echo "            hours or 7200 seconds. One must run 'make unslurmcluster' before changing the time."
	@echo "        You can also combine the flags."
	@echo "        Example: make slurmcluster FLAGS=\"-A input1 -c input2\""
	@echo " "
	@echo "    TF_LOCK"
	@echo "        This can either be \"true\" or \"false\". Default value is 'true'."
	@echo "        Specifies that terraform should not hold a state lock"
	@echo "        during backend migration."
	@echo "        Example: make slurmcluster TF_LOCK=\"true\""
	@echo " "
	@echo "EXAMPLES"
	@echo "    make slurmcluster FLAGS=\"-A -c enroot\" TF_LOCK=\"true\""
	@echo " "



.PHONY: slurmcluster
slurmcluster:
	mkdir -p ~/.slurmcluster
	$(MAKE) -C terraform build 
	./scripts/slurmcluster.sh $(FLAGS)

.PHONY: unslurmcluster
unslurmcluster:
	$(MAKE) -C terraform clean
