.PHONY: slurmcluster
slurmcluster:
	mkdir -p ~/.slurmcluster
	$(MAKE) -C terraform build
	./scripts/slurmcluster.sh

.PHONY: unslurmcluster
unslurmcluster:
	$(MAKE) -C terraform clean
