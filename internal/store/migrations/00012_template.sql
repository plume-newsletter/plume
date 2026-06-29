-- +goose Up
CREATE TABLE template (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,                 -- zero-UUID for prebuilt/global starters
    name       text NOT NULL,
    category   text NOT NULL DEFAULT 'Newsletter',
    body_json  jsonb NOT NULL DEFAULT '[]',
    prebuilt   boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_template_owner ON template (owner_id);
CREATE INDEX ix_template_prebuilt ON template (prebuilt);

-- Seed 3 prebuilt starters (global, owner_id = zero UUID, prebuilt = true).
-- Block field shape (from internal/blocks): heading uses {text, level};
-- text uses {html}; button uses {label, href, align}.
INSERT INTO template (id, owner_id, name, category, body_json, prebuilt) VALUES
('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-000000000000', 'Newsletter', 'Newsletter',
 '[{"id":"b1","type":"heading","text":"This week at your brand","level":2},{"id":"b2","type":"text","html":"A short intro to your update. Tell readers what they will find below."},{"id":"b3","type":"text","html":"Story one — a sentence or two with the key takeaway."},{"id":"b4","type":"button","label":"Read more","href":"https://example.com","align":"left"}]'::jsonb, true),
('00000000-0000-0000-0000-0000000000a2', '00000000-0000-0000-0000-000000000000', 'Product update', 'Product',
 '[{"id":"b1","type":"heading","text":"What''s new","level":2},{"id":"b2","type":"text","html":"We shipped something you''ll like. Here is the headline change in one line."},{"id":"b3","type":"text","html":"Why it matters for you, in plain language."},{"id":"b4","type":"button","label":"Try it now","href":"https://example.com","align":"left"}]'::jsonb, true),
('00000000-0000-0000-0000-0000000000a3', '00000000-0000-0000-0000-000000000000', 'Promo', 'Promo',
 '[{"id":"b1","type":"heading","text":"A little something for you","level":2},{"id":"b2","type":"text","html":"Make the offer clear and time-bound. One sentence."},{"id":"b3","type":"button","label":"Claim offer","href":"https://example.com","align":"center"}]'::jsonb, true);

-- +goose Down
DROP TABLE IF EXISTS template;
