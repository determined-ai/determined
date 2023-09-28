/*
write a SQL migration to create pg table that corresponds to the following go struct.
 
// TaskAlert is the db representation of a task alert.
type TaskAlert struct {
	TaskID    string
	NodeID    string
	DeviceIDs []string
	Action    agentv1.Action // Type?
}
*/

CREATE SEQUENCE task_alerts_id_seq;

CREATE TABLE task_alerts (
    id integer NOT NULL DEFAULT nextval('task_alerts_id_seq'::regclass),
    task_id    text NOT NULL,
    -- do we want the allocation id?
    node_id    text NOT NULL,
    device_ids jsonb,
    action     text NOT NULL
);
