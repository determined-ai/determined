ALTER TABLE experiments
	DROP COLUMN checkpoint_size;
ALTER TABLE experiments
	DROP COLUMN checkpoint_count;
ALTER TABLE trials
	DROP COLUMN checkpoint_size;
ALTER TABLE trials
	DROP COLUMN checkpoint_count;