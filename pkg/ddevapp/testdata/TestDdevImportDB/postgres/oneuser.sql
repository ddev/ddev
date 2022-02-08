--
-- PostgreSQL database dump
--

-- Dumped from database version 14.1 (Debian 14.1-1.pgdg110+1)
-- Dumped by pg_dump version 14.1 (Debian 14.1-1.pgdg110+1)

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
-- Name: users_just_one; Type: TABLE; Schema: public; Owner: db
--

CREATE TABLE public.users_just_one (
    uid integer NOT NULL,
    uuid character varying(128) NOT NULL,
    langcode character varying(12) NOT NULL,
    CONSTRAINT users_uid_check CHECK ((uid >= 0))
);


ALTER TABLE public.users_just_one OWNER TO db;

--
-- Name: TABLE users_just_one; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON TABLE public.users_just_one IS 'The base table for user entities.';


--
-- Name: users_uid_seq; Type: SEQUENCE; Schema: public; Owner: db
--

CREATE SEQUENCE public.users_uid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_uid_seq OWNER TO db;

--
-- Name: users_uid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: db
--

ALTER SEQUENCE public.users_uid_seq OWNED BY public.users_just_one.uid;


--
-- Name: users_just_one uid; Type: DEFAULT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.users_just_one ALTER COLUMN uid SET DEFAULT nextval('public.users_uid_seq'::regclass);


--
-- Data for Name: users_just_one; Type: TABLE DATA; Schema: public; Owner: db
--

COPY public.users_just_one (uid, uuid, langcode) FROM stdin;
0	21ac33ef-45f8-44c2-b766-c4bb2f061a29	en
\.


--
-- Name: users_uid_seq; Type: SEQUENCE SET; Schema: public; Owner: db
--

SELECT pg_catalog.setval('public.users_uid_seq', 7, true);


--
-- Name: users_just_one users____pkey; Type: CONSTRAINT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.users_just_one
    ADD CONSTRAINT users____pkey PRIMARY KEY (uid);


--
-- Name: users_just_one users__user_field__uuid__value__key; Type: CONSTRAINT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.users_just_one
    ADD CONSTRAINT users__user_field__uuid__value__key UNIQUE (uuid);


--
-- PostgreSQL database dump complete
--

