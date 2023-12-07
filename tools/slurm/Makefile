.PHONY: usage
usage:
	@echo "NAME"
	@echo "    make slurmcluster"
	@echo " "
	@echo "SYNOPSIS"
	@echo "    make slurmcluster [FLAGS=\"options\"] [TF_LOCK=\"value\"]"
	@echo " "
	@./scripts/slurmcluster.sh -h


.PHONY: slurmcluster
slurmcluster:
	mkdir -p ~/.slurmcluster
	$(MAKE) -C terraform build 
	./scripts/slurmcluster.sh $(FLAGS)

.PHONY: unslurmcluster
unslurmcluster:
	$(MAKE) -C terraform clean
