CREATE TABLE public.prefixes (
    id uuid PRIMARY KEY DEFAULT extensions.uuid_generate_v4(),
    prefix text NOT NULL UNIQUE,
    is_endpoint boolean DEFAULT false,
    pair_number integer CHECK (pair_number IN (1, 2)),
    description text,
    created_at timestamptz NOT NULL DEFAULT now()
); 