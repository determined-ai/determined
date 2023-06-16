.PHONY: usage
usage:
	@echo "Makefile usage:"
	@echo ""
	@echo " make slurmcluster	- Launches a compute instance with Slurm, Singularity (Apptainer), the Cray"
	@echo "                       Launcher component, and many other dependencies pre-installed. Then, SSH tunnels"
	@echo "                       are opened so that localhost:8081 on your machine points at port 8081 on"
	@echo "                       the compute instance and port 8080 on the compute instance points at"
	@echo "                       localhost:8080 on your machine. Lastly, devcluster is started with the Slurm"
	@echo "                       RM pointed at the remote instance, and local development with devcluster works"
	@echo "                       as always."
	@echo ""
	@echo " make unslurmcluster  - Tears down GCP compute instance made by invoking make slurmcluster."
	@echo ""
	@echo " make slurmcluster [cont=type]  - Invokes a slurmcluster using the specified container run type."
	@echo "									'type' provides a resource_manager.container_run_type value"
	@echo "                                  Options for 'type' are 'singularity' or 'podman'. Default is 'singularity'."
	@echo ""


.PHONY: slurmcluster
slurmcluster:
	mkdir -p ~/.slurmcluster
	$(MAKE) -C terraform build
	- ./scripts/slurmcluster.sh -c $(cont) || ./scripts/slurmcluster.sh -c singularity

.PHONY: unslurmcluster
unslurmcluster:
	$(MAKE) -C terraform clean
