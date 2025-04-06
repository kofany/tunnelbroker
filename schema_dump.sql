--
-- PostgreSQL database dump
--

-- Dumped from database version 15.8
-- Dumped by pg_dump version 15.12 (Debian 15.12-0+deb12u2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- Name: create_relay_entry(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.create_relay_entry() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO ''
    AS $$
DECLARE
    new_hex_code text;
BEGIN
    -- Generate a unique 4-character hex code
    LOOP
        new_hex_code := lpad(to_hex(floor(random() * 65536)::integer), 4, '0'); -- Generate hex code

        -- Check if the hex code is unique
        IF NOT EXISTS (SELECT 1 FROM public.users WHERE id = new_hex_code) THEN
            EXIT; -- Exit the loop if the hex code is unique
        END IF;
    END LOOP;

    -- Insert into public.relay
    INSERT INTO public.relay (auth_user_id, generated_hex4)
    VALUES (NEW.id, new_hex_code);

    RETURN NEW;
END;
$$;


--
-- Name: delete_relay(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_relay() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO ''
    AS $$BEGIN
    -- Delete from public.relay
    DELETE FROM public.relay WHERE auth_user_id = OLD.id;

    RETURN OLD;
END;$$;


--
-- Name: get_next_ula_number(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_next_ula_number() RETURNS bigint
    LANGUAGE plpgsql
    SET search_path TO 'public'
    AS $$
BEGIN
    RETURN nextval('ula_seq');
END;
$$;


--
-- Name: init_tunnels_table(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.init_tunnels_table() RETURNS void
    LANGUAGE plpgsql
    SET search_path TO 'public'
    AS $$
BEGIN
    -- Enable RLS
    ALTER TABLE tunnels ENABLE ROW LEVEL SECURITY;

    -- Create policies
    CREATE POLICY "Enable read access for tunnel owners" ON tunnels
        FOR SELECT USING (auth.uid()::text = user_id::text);

    CREATE POLICY "Enable insert for tunnel owners" ON tunnels
        FOR INSERT WITH CHECK (auth.uid()::text = user_id::text);

    CREATE POLICY "Enable update for tunnel owners" ON tunnels
        FOR UPDATE USING (auth.uid()::text = user_id::text);

    CREATE POLICY "Enable delete for tunnel owners" ON tunnels
        FOR DELETE USING (auth.uid()::text = user_id::text);
END;
$$;


--
-- Name: init_users_table(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.init_users_table() RETURNS void
    LANGUAGE plpgsql
    SET search_path TO 'public'
    AS $$
BEGIN
    -- Enable RLS
    ALTER TABLE users ENABLE ROW LEVEL SECURITY;

    -- Create policies
    CREATE POLICY "Enable read access for all users" ON users
        FOR SELECT USING (true);

    CREATE POLICY "Enable insert for authenticated users only" ON users
        FOR INSERT WITH CHECK (auth.role() = 'authenticated');

    CREATE POLICY "Enable update for users based on id" ON users
        FOR UPDATE USING (auth.uid() = id);

    CREATE POLICY "Enable delete for users based on id" ON users
        FOR DELETE USING (auth.uid() = id);
END;
$$;


--
-- Name: insert_user_from_relay(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.insert_user_from_relay() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path TO ''
    AS $$
BEGIN
    INSERT INTO public.users (id, created_at)
    VALUES (NEW.generated_hex4, now());
    RETURN NEW;
END;
$$;


--
-- Name: update_tunnels_created(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_tunnels_created() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path TO 'public'
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE users 
        SET tunnels_created = tunnels_created + 1
        WHERE id = NEW.user_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE users 
        SET tunnels_created = tunnels_created - 1
        WHERE id = OLD.user_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path TO 'public'
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: relay; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.relay (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_user_id uuid NOT NULL,
    generated_hex4 text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: tunnels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tunnels (
    id text NOT NULL,
    user_id text NOT NULL,
    type text NOT NULL,
    status text NOT NULL,
    server_ipv4 text NOT NULL,
    client_ipv4 text NOT NULL,
    endpoint_local text NOT NULL,
    endpoint_remote text NOT NULL,
    delegated_prefix_1 text NOT NULL,
    delegated_prefix_2 text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    delegated_prefix_3 text,
    CONSTRAINT tunnels_status_check CHECK ((status = ANY (ARRAY['active'::text, 'suspended'::text]))),
    CONSTRAINT tunnels_type_check CHECK ((type = ANY (ARRAY['sit'::text, 'gre'::text])))
);


--
-- Name: COLUMN tunnels.delegated_prefix_3; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tunnels.delegated_prefix_3 IS 'Third delegated /64 prefix from dedicated /48 range';


--
-- Name: ula_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ula_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id text NOT NULL,
    created_tunnels integer DEFAULT 0,
    active_tunnels integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: relay relay_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.relay
    ADD CONSTRAINT relay_pkey PRIMARY KEY (id);


--
-- Name: tunnels tunnels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tunnels
    ADD CONSTRAINT tunnels_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_tunnels_delegated_prefix_3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tunnels_delegated_prefix_3 ON public.tunnels USING btree (delegated_prefix_3);


--
-- Name: idx_tunnels_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tunnels_status ON public.tunnels USING btree (status);


--
-- Name: idx_tunnels_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tunnels_user_id ON public.tunnels USING btree (user_id);


--
-- Name: relay after_relay_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER after_relay_insert AFTER INSERT ON public.relay FOR EACH ROW EXECUTE FUNCTION public.insert_user_from_relay();


--
-- Name: relay Allow delete for authenticated users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow delete for authenticated users" ON public.relay FOR DELETE TO authenticated USING (true);


--
-- Name: tunnels Allow delete for tunnel owners; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow delete for tunnel owners" ON public.tunnels FOR DELETE TO authenticated USING (((( SELECT auth.uid() AS uid))::text = user_id));


--
-- Name: users Allow delete for users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow delete for users" ON public.users FOR DELETE TO authenticated USING ((( SELECT (auth.uid())::text AS uid) = id));


--
-- Name: relay Allow insert for authenticated users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow insert for authenticated users" ON public.relay FOR INSERT TO authenticated WITH CHECK ((auth.uid() IS NOT NULL));


--
-- Name: tunnels Allow insert for tunnel owners; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow insert for tunnel owners" ON public.tunnels FOR INSERT TO authenticated WITH CHECK (((( SELECT auth.uid() AS uid))::text = user_id));


--
-- Name: users Allow insert for users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow insert for users" ON public.users FOR INSERT TO authenticated WITH CHECK (( SELECT (auth.uid() IS NOT NULL)));


--
-- Name: relay Allow select for authenticated users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow select for authenticated users" ON public.relay FOR SELECT TO authenticated USING (true);


--
-- Name: tunnels Allow select for tunnel owners; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow select for tunnel owners" ON public.tunnels FOR SELECT TO authenticated USING (((( SELECT auth.uid() AS uid))::text = user_id));


--
-- Name: users Allow select for users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow select for users" ON public.users FOR SELECT TO authenticated USING ((( SELECT (auth.uid())::text AS uid) = id));


--
-- Name: relay Allow update for authenticated users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow update for authenticated users" ON public.relay FOR UPDATE TO authenticated USING (true) WITH CHECK ((auth.uid() IS NOT NULL));


--
-- Name: tunnels Allow update for tunnel owners; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow update for tunnel owners" ON public.tunnels FOR UPDATE TO authenticated USING (((( SELECT auth.uid() AS uid))::text = user_id));


--
-- Name: users Allow update for users; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY "Allow update for users" ON public.users FOR UPDATE TO authenticated USING ((( SELECT (auth.uid())::text AS uid) = id)) WITH CHECK (( SELECT (auth.uid() IS NOT NULL)));


--
-- Name: relay; Type: ROW SECURITY; Schema: public; Owner: -
--

ALTER TABLE public.relay ENABLE ROW LEVEL SECURITY;

--
-- Name: tunnels; Type: ROW SECURITY; Schema: public; Owner: -
--

ALTER TABLE public.tunnels ENABLE ROW LEVEL SECURITY;

--
-- Name: users; Type: ROW SECURITY; Schema: public; Owner: -
--

ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;

--
-- PostgreSQL database dump complete
--

--
-- PostgreSQL database dump
--

-- Dumped from database version 15.8
-- Dumped by pg_dump version 15.12 (Debian 15.12-0+deb12u2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: users; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.users (
    instance_id uuid,
    id uuid NOT NULL,
    aud character varying(255),
    role character varying(255),
    email character varying(255),
    encrypted_password character varying(255),
    email_confirmed_at timestamp with time zone,
    invited_at timestamp with time zone,
    confirmation_token character varying(255),
    confirmation_sent_at timestamp with time zone,
    recovery_token character varying(255),
    recovery_sent_at timestamp with time zone,
    email_change_token_new character varying(255),
    email_change character varying(255),
    email_change_sent_at timestamp with time zone,
    last_sign_in_at timestamp with time zone,
    raw_app_meta_data jsonb,
    raw_user_meta_data jsonb,
    is_super_admin boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    phone text DEFAULT NULL::character varying,
    phone_confirmed_at timestamp with time zone,
    phone_change text DEFAULT ''::character varying,
    phone_change_token character varying(255) DEFAULT ''::character varying,
    phone_change_sent_at timestamp with time zone,
    confirmed_at timestamp with time zone GENERATED ALWAYS AS (LEAST(email_confirmed_at, phone_confirmed_at)) STORED,
    email_change_token_current character varying(255) DEFAULT ''::character varying,
    email_change_confirm_status smallint DEFAULT 0,
    banned_until timestamp with time zone,
    reauthentication_token character varying(255) DEFAULT ''::character varying,
    reauthentication_sent_at timestamp with time zone,
    is_sso_user boolean DEFAULT false NOT NULL,
    deleted_at timestamp with time zone,
    is_anonymous boolean DEFAULT false NOT NULL,
    CONSTRAINT users_email_change_confirm_status_check CHECK (((email_change_confirm_status >= 0) AND (email_change_confirm_status <= 2)))
);


--
-- Name: TABLE users; Type: COMMENT; Schema: auth; Owner: -
--

COMMENT ON TABLE auth.users IS 'Auth: Stores user login data within a secure schema.';


--
-- Name: COLUMN users.is_sso_user; Type: COMMENT; Schema: auth; Owner: -
--

COMMENT ON COLUMN auth.users.is_sso_user IS 'Auth: Set this column to true when the account comes from SSO. These accounts can have duplicate emails.';


--
-- Name: users users_phone_key; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.users
    ADD CONSTRAINT users_phone_key UNIQUE (phone);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: confirmation_token_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX confirmation_token_idx ON auth.users USING btree (confirmation_token) WHERE ((confirmation_token)::text !~ '^[0-9 ]*$'::text);


--
-- Name: email_change_token_current_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX email_change_token_current_idx ON auth.users USING btree (email_change_token_current) WHERE ((email_change_token_current)::text !~ '^[0-9 ]*$'::text);


--
-- Name: email_change_token_new_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX email_change_token_new_idx ON auth.users USING btree (email_change_token_new) WHERE ((email_change_token_new)::text !~ '^[0-9 ]*$'::text);


--
-- Name: reauthentication_token_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX reauthentication_token_idx ON auth.users USING btree (reauthentication_token) WHERE ((reauthentication_token)::text !~ '^[0-9 ]*$'::text);


--
-- Name: recovery_token_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX recovery_token_idx ON auth.users USING btree (recovery_token) WHERE ((recovery_token)::text !~ '^[0-9 ]*$'::text);


--
-- Name: users_email_partial_key; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX users_email_partial_key ON auth.users USING btree (email) WHERE (is_sso_user = false);


--
-- Name: INDEX users_email_partial_key; Type: COMMENT; Schema: auth; Owner: -
--

COMMENT ON INDEX auth.users_email_partial_key IS 'Auth: A partial unique index that applies only when is_sso_user is false';


--
-- Name: users_instance_id_email_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX users_instance_id_email_idx ON auth.users USING btree (instance_id, lower((email)::text));


--
-- Name: users_instance_id_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX users_instance_id_idx ON auth.users USING btree (instance_id);


--
-- Name: users_is_anonymous_idx; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX users_is_anonymous_idx ON auth.users USING btree (is_anonymous);


--
-- Name: users after_user_creation; Type: TRIGGER; Schema: auth; Owner: -
--

CREATE TRIGGER after_user_creation AFTER INSERT ON auth.users FOR EACH ROW EXECUTE FUNCTION public.create_relay_entry();


--
-- Name: users after_user_deletion; Type: TRIGGER; Schema: auth; Owner: -
--

CREATE TRIGGER after_user_deletion AFTER DELETE ON auth.users FOR EACH ROW EXECUTE FUNCTION public.delete_relay();


--
-- Name: users; Type: ROW SECURITY; Schema: auth; Owner: -
--

ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;

--
-- PostgreSQL database dump complete
--

