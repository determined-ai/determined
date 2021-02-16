UPDATE experiment_snapshots
SET content = content #- '{searcher_state,search_method_state,waiting_checkpoints}';