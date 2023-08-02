SELECT
    array_to_json(
        array_remove(
            array(
                SELECT DISTINCT allocation_id
                FROM
                    task_logs
                WHERE
                    task_id = $1
            ),
            NULL
        )
    ) AS allocation_ids,
    array_to_json(
        array_remove(
            array(
                SELECT DISTINCT agent_id
                FROM
                    task_logs
                WHERE
                    task_id = $1
            ),
            NULL
        )
    ) AS agent_ids,
    array_to_json(
        array_remove(
            array(
                SELECT DISTINCT container_id
                FROM
                    task_logs
                WHERE
                    task_id = $1
            ),
            NULL
        )
    ) AS container_ids,
    array_to_json(
        array_remove(
            array(
                SELECT DISTINCT rank_id
                FROM
                    task_logs
                WHERE
                    task_id = $1
            ),
            NULL
        )
    ) AS rank_ids,
    array_to_json(
        array_remove(
            array(
                SELECT DISTINCT stdtype
                FROM
                    task_logs
                WHERE
                    task_id = $1
            ),
            NULL
        )
    ) AS stdtypes,
    array_to_json(
        array_remove(
            array(
                SELECT DISTINCT source
                FROM
                    task_logs
                WHERE
                    task_id = $1
            ),
            NULL
        )
    ) AS sources;
