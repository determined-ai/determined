select distinct
                agent_id,
                container_id,
                rank_id,
                std_type,
                source
from trial_logs where trial_id = $1;
