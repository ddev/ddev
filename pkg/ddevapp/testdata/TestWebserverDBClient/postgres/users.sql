--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.24
-- Dumped by pg_dump version 9.6.24

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

SET default_with_oids = false;

--
-- Name: users; Type: TABLE; Schema: public; Owner: db
--

CREATE TABLE public.users (
    uid bigint DEFAULT 0 NOT NULL,
    name character varying(60) DEFAULT ''::character varying NOT NULL,
    pass character varying(128) DEFAULT ''::character varying NOT NULL,
    mail character varying(254) DEFAULT ''::character varying,
    theme character varying(255) DEFAULT ''::character varying NOT NULL,
    signature character varying(255) DEFAULT ''::character varying NOT NULL,
    signature_format character varying(255),
    created integer DEFAULT 0 NOT NULL,
    changed integer DEFAULT 0 NOT NULL,
    access integer DEFAULT 0 NOT NULL,
    login integer DEFAULT 0 NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    timezone character varying(32),
    language character varying(12) DEFAULT ''::character varying NOT NULL,
    picture integer DEFAULT 0 NOT NULL,
    init character varying(254) DEFAULT ''::character varying,
    data bytea,
    CONSTRAINT users_uid_check CHECK ((uid >= 0))
);


ALTER TABLE public.users OWNER TO db;

--
-- Name: TABLE users; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON TABLE public.users IS 'Stores user data.';


--
-- Name: COLUMN users.uid; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.uid IS 'Primary Key: Unique user ID.';


--
-- Name: COLUMN users.name; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.name IS 'Unique user name.';


--
-- Name: COLUMN users.pass; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.pass IS 'User''s password (hashed).';


--
-- Name: COLUMN users.mail; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.mail IS 'User''s e-mail address.';


--
-- Name: COLUMN users.theme; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.theme IS 'User''s default theme.';


--
-- Name: COLUMN users.signature; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.signature IS 'User''s signature.';


--
-- Name: COLUMN users.signature_format; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.signature_format IS 'The filter_format.format of the signature.';


--
-- Name: COLUMN users.created; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.created IS 'Timestamp for when user was created.';


--
-- Name: COLUMN users.changed; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.changed IS 'Timestamp for when user was changed.';


--
-- Name: COLUMN users.access; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.access IS 'Timestamp for previous time user accessed the site.';


--
-- Name: COLUMN users.login; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.login IS 'Timestamp for user''s last login.';


--
-- Name: COLUMN users.status; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.status IS 'Whether the user is active(1) or blocked(0).';


--
-- Name: COLUMN users.timezone; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.timezone IS 'User''s time zone.';


--
-- Name: COLUMN users.language; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.language IS 'User''s default language.';


--
-- Name: COLUMN users.picture; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.picture IS 'Foreign key: file_managed.fid of user''s picture.';


--
-- Name: COLUMN users.init; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.init IS 'E-mail address used for initial account creation.';


--
-- Name: COLUMN users.data; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON COLUMN public.users.data IS 'A serialized array of name value pairs that are related to the user. Any form values posted during user edit are stored and are loaded into the $user object during user_load(). Use of this field is discouraged and it will likely disappear in a future version of Drupal.';


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: db
--

COPY public.users (uid, name, pass, mail, theme, signature, signature_format, created, changed, access, login, status, timezone, language, picture, init, data) FROM stdin;
0						\N	0	0	0	0	0	\N		0		\N
1	admin	$S$DNcYx6Inh./nX3YGehhp3cSg0pH9ZtqeAlqqwGjThoNsa9UXPlyh	admin@example.com			\N	1644002730	1644002786	1644003023	1644002786	1	America/Los_Angeles		0	admin@example.com	\\x623a303b
\.


--
-- Name: users users_name_key; Type: CONSTRAINT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_name_key UNIQUE (name);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (uid);


--
-- Name: users_access_idx; Type: INDEX; Schema: public; Owner: db
--

CREATE INDEX users_access_idx ON public.users USING btree (access);


--
-- Name: users_changed_idx; Type: INDEX; Schema: public; Owner: db
--

CREATE INDEX users_changed_idx ON public.users USING btree (changed);


--
-- Name: users_created_idx; Type: INDEX; Schema: public; Owner: db
--

CREATE INDEX users_created_idx ON public.users USING btree (created);


--
-- Name: users_mail_idx; Type: INDEX; Schema: public; Owner: db
--

CREATE INDEX users_mail_idx ON public.users USING btree (mail);


--
-- Name: users_picture_idx; Type: INDEX; Schema: public; Owner: db
--

CREATE INDEX users_picture_idx ON public.users USING btree (picture);


--
-- PostgreSQL database dump complete
--

