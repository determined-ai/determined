select array(
           select distinct agent_id
           from trial_logs where trial_id = $1
           ) as agent_ids,
       array(
           select distinct container_id
           from trial_logs where trial_id = $1
       ) as container_ids,
       array(
           select distinct rank_id
           from trial_logs where trial_id = $1
       ) as rank_ids,
       array(
           select distinct stdtype
           from trial_logs where trial_id = $1
       ) as stdtypes,
       array(
           select distinct source
           from trial_logs where trial_id = $1
       ) as sources;
