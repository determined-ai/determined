UPDATE experiments
SET config = jsonb_insert(config, '{slurm,sbatch_args}', config->'environment'->'slurm')
WHERE config->'environment'->'slurm' IS NOT NULL AND NOT config->'environment'->'slurm' = '[]';

UPDATE experiments
SET config = config #- '{environment,slurm}';
