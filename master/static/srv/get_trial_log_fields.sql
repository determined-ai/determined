SELECT array_to_json(array_remove(array(
               SELECT DISTINCT agent_id
               FROM trial_logs WHERE trial_id = $1
           ), NULL)) AS agent_ids,
       array_to_json(array_remove(array(
               SELECT DISTINCT container_id
               FROM trial_logs WHERE trial_id = $1
           ), NULL)) AS container_ids,
       array_to_json(array_remove(array(
               SELECT DISTINCT rank_id
               FROM trial_logs WHERE trial_id = $1
           ), NULL)) AS rank_ids,
       array_to_json(array_remove(array(
               SELECT DISTINCT stdtype
               FROM trial_logs WHERE trial_id = $1
           ), NULL)) AS stdtypes,
       array_to_json(array_remove(array(
               SELECT DISTINCT source
               FROM trial_logs WHERE trial_id = $1
           ), NULL)) AS sources;
