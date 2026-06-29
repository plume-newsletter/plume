-- +goose Up
CREATE UNIQUE INDEX ux_field_value_subscriber_field ON subscriber_field_value (subscriber_id, custom_field_id);

-- +goose Down
DROP INDEX IF EXISTS ux_field_value_subscriber_field;
