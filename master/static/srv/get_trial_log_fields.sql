select array_to_json(array_remove(array(
               select distinct agent_id
               from trial_logs where trial_id = $1
           ), NULL)) as agent_ids,
       array_to_json(array_remove(array(
               select distinct container_id
               from trial_logs where trial_id = $1
           ), NULL)) as container_ids,
       array_to_json(array_remove(array(
               select distinct rank_id
               from trial_logs where trial_id = $1
           ), NULL)) as rank_ids,
       array_to_json(array_remove(array(
               select distinct stdtype
               from trial_logs where trial_id = $1
           ), NULL)) as stdtypes,
       array_to_json(array_remove(array(
               select distinct source
               from trial_logs where trial_id = $1
           ), NULL)) as sources;
