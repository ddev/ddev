--
-- PostgreSQL database dump
--

-- Dumped from database version 10.19 (Debian 10.19-1.pgdg90+1)
-- Dumped by pg_dump version 10.19 (Debian 10.19-1.pgdg90+1)

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
-- Name: stdintable; Type: TABLE; Schema: public; Owner: db
--

CREATE TABLE public.stdintable (
    uid integer NOT NULL,
    uuid character varying(128) NOT NULL,
    langcode character varying(12) NOT NULL,
    CONSTRAINT users_uid_check CHECK ((uid >= 0))
);


ALTER TABLE public.stdintable OWNER TO db;

--
-- Name: TABLE stdintable; Type: COMMENT; Schema: public; Owner: db
--

COMMENT ON TABLE public.stdintable IS 'The base table for user entities.';


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

ALTER SEQUENCE public.users_uid_seq OWNED BY public.stdintable.uid;


--
-- Name: stdintable uid; Type: DEFAULT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.stdintable ALTER COLUMN uid SET DEFAULT nextval('public.users_uid_seq'::regclass);


--
-- Data for Name: stdintable; Type: TABLE DATA; Schema: public; Owner: db
--

COPY public.stdintable (uid, uuid, langcode) FROM stdin;
0	e6b6f38b-3133-474a-b1b0-35a6459ef010	en
1	6c3ec872-f73e-4074-bfd3-029633935e26	en
\.


--
-- Name: users_uid_seq; Type: SEQUENCE SET; Schema: public; Owner: db
--

SELECT pg_catalog.setval('public.users_uid_seq', 7, true);


--
-- Name: stdintable users____pkey; Type: CONSTRAINT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.stdintable
    ADD CONSTRAINT users____pkey PRIMARY KEY (uid);


--
-- Name: stdintable users__user_field__uuid__value__key; Type: CONSTRAINT; Schema: public; Owner: db
--

ALTER TABLE ONLY public.stdintable
    ADD CONSTRAINT users__user_field__uuid__value__key UNIQUE (uuid);


--
-- PostgreSQL database dump complete
--

