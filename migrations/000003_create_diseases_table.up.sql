CREATE TABLE "diseases" (
  "id" uuid PRIMARY KEY DEFAULT (uuid_generate_v4()),
  "alias" varchar NOT NULL UNIQUE, -- matches AI label (e.g., rice_blast)
  "name" varchar NOT NULL,
  "category" varchar NOT NULL,
  "image_url" varchar NOT NULL,
  "description" text NOT NULL,
  "spread_details" text NOT NULL,
  "match_weather" jsonb DEFAULT '[]',
  "symptoms" jsonb DEFAULT '[]',
  "prevention" jsonb DEFAULT '[]',
  "treatment" jsonb DEFAULT '[]',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX "idx_diseases_alias" ON "diseases" ("alias");
