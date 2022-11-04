UPDATE experiments
SET config = jsonb_insert(config, '{environment,slurm}', config->'slurm'->'sbatch_args')
WHERE config->'slurm'->'sbatch_args' IS NOT NULL AND NOT config->'slurm'->'sbatch_args' = '[]';

UPDATE experiments
SET config = config #- '{slurm}';

UPDATE experiments
SET config = config #- '{pbs}';
