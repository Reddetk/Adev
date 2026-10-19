-- An agent config belongs to a target system. Without this constraint a PUT for a
-- target system that does not exist silently created an orphan row.

DELETE FROM agent_configs
WHERE target_system_id NOT IN (SELECT id FROM target_systems);

ALTER TABLE agent_configs
    ADD CONSTRAINT agent_configs_target_system_fk
    FOREIGN KEY (target_system_id) REFERENCES target_systems (id) ON DELETE CASCADE;
