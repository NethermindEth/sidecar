--
-- PostgreSQL database dump
--

-- Dumped from database version 14.11 (Homebrew)
-- Dumped by pg_dump version 15.7 (Homebrew)

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

-- *not* creating schema, since initdb creates it


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: active_reward_for_all; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_reward_for_all (
    avs character varying,
    reward_hash character varying,
    token character varying,
    amount numeric,
    strategy character varying,
    multiplier numeric,
    strategy_index bigint,
    block_number bigint,
    start_timestamp timestamp without time zone,
    end_timestamp timestamp without time zone,
    duration bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: active_reward_for_all_submissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_reward_for_all_submissions (
    id integer NOT NULL,
    avs character varying,
    reward_hash character varying,
    token character varying,
    amount numeric,
    strategy character varying,
    multiplier numeric,
    strategy_index bigint,
    transaction_hash character varying,
    log_index bigint,
    block_number bigint,
    start_timestamp timestamp without time zone,
    end_timestamp timestamp without time zone,
    duration bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: active_reward_for_all_submissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.active_reward_for_all_submissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: active_reward_for_all_submissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.active_reward_for_all_submissions_id_seq OWNED BY public.active_reward_for_all_submissions.id;


--
-- Name: active_reward_submissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_reward_submissions (
    id integer NOT NULL,
    avs character varying,
    reward_hash character varying,
    token character varying,
    amount numeric,
    strategy character varying,
    multiplier numeric,
    strategy_index bigint,
    transaction_hash character varying,
    log_index bigint,
    block_number bigint,
    start_timestamp timestamp without time zone,
    end_timestamp timestamp without time zone,
    duration bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: active_reward_submissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.active_reward_submissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: active_reward_submissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.active_reward_submissions_id_seq OWNED BY public.active_reward_submissions.id;


--
-- Name: active_rewards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_rewards (
    avs character varying,
    reward_hash character varying,
    token character varying,
    amount numeric,
    strategy character varying,
    multiplier numeric,
    strategy_index bigint,
    block_number bigint,
    start_timestamp timestamp without time zone,
    end_timestamp timestamp without time zone,
    duration bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: avs_operator_changes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.avs_operator_changes (
    id integer NOT NULL,
    operator character varying,
    avs character varying,
    registered boolean,
    transaction_hash character varying,
    log_index bigint,
    block_number bigint
);


--
-- Name: avs_operator_changes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.avs_operator_changes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: avs_operator_changes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.avs_operator_changes_id_seq OWNED BY public.avs_operator_changes.id;


--
-- Name: blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blocks (
    id integer NOT NULL,
    number bigint NOT NULL,
    hash character varying(255) NOT NULL,
    blob_path text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    block_time timestamp with time zone NOT NULL
);


--
-- Name: block_sequences_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.block_sequences_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: block_sequences_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.block_sequences_id_seq OWNED BY public.blocks.id;


--
-- Name: contracts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.contracts (
    contract_address character varying(255) NOT NULL,
    contract_abi text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    bytecode_hash character varying(64) DEFAULT NULL::character varying,
    verified boolean DEFAULT false,
    matching_contract_address character varying(255) DEFAULT NULL::character varying,
    checked_for_proxy boolean DEFAULT false NOT NULL,
    id integer NOT NULL,
    checked_for_abi boolean
);


--
-- Name: contracts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.contracts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: contracts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.contracts_id_seq OWNED BY public.contracts.id;


--
-- Name: delegated_stakers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.delegated_stakers (
    staker character varying,
    operator character varying,
    block_number bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.migrations (
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone
);


--
-- Name: operator_restaked_strategies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.operator_restaked_strategies (
    id integer NOT NULL,
    block_number bigint NOT NULL,
    operator character varying NOT NULL,
    avs character varying NOT NULL,
    strategy character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    block_time timestamp with time zone NOT NULL,
    avs_directory_address character varying
);


--
-- Name: operator_restaked_strategies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.operator_restaked_strategies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: operator_restaked_strategies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.operator_restaked_strategies_id_seq OWNED BY public.operator_restaked_strategies.id;


--
-- Name: operator_share_changes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.operator_share_changes (
    id integer NOT NULL,
    operator character varying,
    strategy character varying,
    shares numeric,
    transaction_hash character varying,
    log_index bigint,
    block_number bigint
);


--
-- Name: operator_share_changes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.operator_share_changes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: operator_share_changes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.operator_share_changes_id_seq OWNED BY public.operator_share_changes.id;


--
-- Name: operator_shares; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.operator_shares (
    operator character varying,
    strategy character varying,
    shares numeric,
    block_number bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: proxy_contracts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.proxy_contracts (
    block_number bigint NOT NULL,
    contract_address character varying(255) NOT NULL,
    proxy_contract_address character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: registered_avs_operators; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.registered_avs_operators (
    operator character varying,
    avs character varying,
    block_number bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: staker_delegation_changes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staker_delegation_changes (
    id integer NOT NULL,
    staker character varying,
    operator character varying,
    delegated boolean,
    transaction_hash character varying,
    log_index bigint,
    block_number bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: staker_delegation_changes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.staker_delegation_changes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: staker_delegation_changes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.staker_delegation_changes_id_seq OWNED BY public.staker_delegation_changes.id;


--
-- Name: staker_share_changes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staker_share_changes (
    id integer NOT NULL,
    staker character varying,
    strategy character varying,
    shares numeric,
    transaction_hash character varying,
    log_index bigint,
    block_number bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: staker_share_changes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.staker_share_changes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: staker_share_changes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.staker_share_changes_id_seq OWNED BY public.staker_share_changes.id;


--
-- Name: staker_shares; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staker_shares (
    staker character varying,
    strategy character varying,
    shares numeric,
    block_number bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: transaction_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transaction_logs (
    transaction_hash character varying(255) NOT NULL,
    address character varying(255) NOT NULL,
    arguments jsonb,
    event_name character varying(255) NOT NULL,
    log_index bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    block_number bigint NOT NULL,
    block_sequence_id bigint NOT NULL,
    transaction_index integer NOT NULL,
    output_data jsonb
);


--
-- Name: transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transactions (
    block_number bigint NOT NULL,
    transaction_hash character varying(255) NOT NULL,
    transaction_index bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    from_address character varying(255) NOT NULL,
    to_address character varying(255) DEFAULT NULL::character varying,
    block_sequence_id bigint NOT NULL,
    contract_address character varying(255) DEFAULT NULL::character varying,
    bytecode_hash character varying(64) DEFAULT NULL::character varying
);


--
-- Name: unverified_contracts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.unverified_contracts (
    contract_address character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: active_reward_for_all_submissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_reward_for_all_submissions ALTER COLUMN id SET DEFAULT nextval('public.active_reward_for_all_submissions_id_seq'::regclass);


--
-- Name: active_reward_submissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_reward_submissions ALTER COLUMN id SET DEFAULT nextval('public.active_reward_submissions_id_seq'::regclass);


--
-- Name: avs_operator_changes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.avs_operator_changes ALTER COLUMN id SET DEFAULT nextval('public.avs_operator_changes_id_seq'::regclass);


--
-- Name: blocks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blocks ALTER COLUMN id SET DEFAULT nextval('public.block_sequences_id_seq'::regclass);


--
-- Name: contracts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.contracts ALTER COLUMN id SET DEFAULT nextval('public.contracts_id_seq'::regclass);


--
-- Name: operator_restaked_strategies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.operator_restaked_strategies ALTER COLUMN id SET DEFAULT nextval('public.operator_restaked_strategies_id_seq'::regclass);


--
-- Name: operator_share_changes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.operator_share_changes ALTER COLUMN id SET DEFAULT nextval('public.operator_share_changes_id_seq'::regclass);


--
-- Name: staker_delegation_changes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staker_delegation_changes ALTER COLUMN id SET DEFAULT nextval('public.staker_delegation_changes_id_seq'::regclass);


--
-- Name: staker_share_changes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staker_share_changes ALTER COLUMN id SET DEFAULT nextval('public.staker_share_changes_id_seq'::regclass);


--
-- Name: active_reward_for_all_submissions active_reward_for_all_submissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_reward_for_all_submissions
    ADD CONSTRAINT active_reward_for_all_submissions_pkey PRIMARY KEY (id);


--
-- Name: active_reward_submissions active_reward_submissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_reward_submissions
    ADD CONSTRAINT active_reward_submissions_pkey PRIMARY KEY (id);


--
-- Name: avs_operator_changes avs_operator_changes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.avs_operator_changes
    ADD CONSTRAINT avs_operator_changes_pkey PRIMARY KEY (id);


--
-- Name: blocks block_sequences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT block_sequences_pkey PRIMARY KEY (id);


--
-- Name: blocks blocks_unique_block_number_hash; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_unique_block_number_hash UNIQUE (number, hash);


--
-- Name: contracts contracts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_pkey PRIMARY KEY (contract_address);


--
-- Name: delegated_stakers delegated_stakers_staker_operator_block_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.delegated_stakers
    ADD CONSTRAINT delegated_stakers_staker_operator_block_number_key UNIQUE (staker, operator, block_number);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (name);


--
-- Name: operator_restaked_strategies operator_restaked_strategies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.operator_restaked_strategies
    ADD CONSTRAINT operator_restaked_strategies_pkey PRIMARY KEY (id);


--
-- Name: operator_share_changes operator_share_changes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.operator_share_changes
    ADD CONSTRAINT operator_share_changes_pkey PRIMARY KEY (id);


--
-- Name: operator_shares operator_shares_operator_strategy_block_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.operator_shares
    ADD CONSTRAINT operator_shares_operator_strategy_block_number_key UNIQUE (operator, strategy, block_number);


--
-- Name: registered_avs_operators registered_avs_operators_operator_avs_block_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registered_avs_operators
    ADD CONSTRAINT registered_avs_operators_operator_avs_block_number_key UNIQUE (operator, avs, block_number);


--
-- Name: staker_delegation_changes staker_delegation_changes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staker_delegation_changes
    ADD CONSTRAINT staker_delegation_changes_pkey PRIMARY KEY (id);


--
-- Name: staker_share_changes staker_share_changes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staker_share_changes
    ADD CONSTRAINT staker_share_changes_pkey PRIMARY KEY (id);


--
-- Name: staker_shares staker_shares_staker_strategy_block_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staker_shares
    ADD CONSTRAINT staker_shares_staker_strategy_block_number_key UNIQUE (staker, strategy, block_number);


--
-- Name: transactions transactions_transaction_hash_sequence_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_transaction_hash_sequence_id_key UNIQUE (transaction_hash, block_sequence_id);


--
-- Name: unverified_contracts unverified_contracts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unverified_contracts
    ADD CONSTRAINT unverified_contracts_pkey PRIMARY KEY (contract_address);


--
-- Name: idx_active_reward_for_all_avs; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_reward_for_all_avs ON public.active_reward_for_all USING btree (avs);


--
-- Name: idx_active_reward_for_all_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_reward_for_all_block ON public.active_reward_for_all USING btree (block_number);


--
-- Name: idx_active_reward_for_all_submissions_avs; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_reward_for_all_submissions_avs ON public.active_reward_for_all_submissions USING btree (avs);


--
-- Name: idx_active_reward_for_all_submissions_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_reward_for_all_submissions_block ON public.active_reward_for_all_submissions USING btree (block_number);


--
-- Name: idx_active_reward_submissions_avs; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_reward_submissions_avs ON public.active_reward_submissions USING btree (avs);


--
-- Name: idx_active_reward_submissions_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_reward_submissions_block ON public.active_reward_submissions USING btree (block_number);


--
-- Name: idx_active_rewards_avs; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_rewards_avs ON public.active_rewards USING btree (avs);


--
-- Name: idx_active_rewards_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_active_rewards_block ON public.active_rewards USING btree (block_number);


--
-- Name: idx_avs_operator_changes_avs_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_avs_operator_changes_avs_operator ON public.avs_operator_changes USING btree (avs, operator);


--
-- Name: idx_avs_operator_changes_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_avs_operator_changes_block ON public.avs_operator_changes USING btree (block_number);


--
-- Name: idx_bytecode_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_bytecode_hash ON public.contracts USING btree (bytecode_hash);


--
-- Name: idx_delegated_stakers_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_delegated_stakers_block ON public.delegated_stakers USING btree (block_number);


--
-- Name: idx_delegated_stakers_staker_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_delegated_stakers_staker_operator ON public.delegated_stakers USING btree (staker, operator);


--
-- Name: idx_operator_share_changes_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_operator_share_changes_block ON public.operator_share_changes USING btree (block_number);


--
-- Name: idx_operator_share_changes_operator_strat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_operator_share_changes_operator_strat ON public.operator_share_changes USING btree (operator, strategy);


--
-- Name: idx_operator_shares_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_operator_shares_block ON public.operator_shares USING btree (block_number);


--
-- Name: idx_operator_shares_operator_strategy; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_operator_shares_operator_strategy ON public.operator_shares USING btree (operator, strategy);


--
-- Name: idx_proxy_contract_contract_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_proxy_contract_contract_address ON public.proxy_contracts USING btree (contract_address);


--
-- Name: idx_proxy_contract_proxy_contract_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_proxy_contract_proxy_contract_address ON public.proxy_contracts USING btree (proxy_contract_address);


--
-- Name: idx_registered_avs_operators_avs_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_registered_avs_operators_avs_operator ON public.registered_avs_operators USING btree (avs, operator);


--
-- Name: idx_registered_avs_operators_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_registered_avs_operators_block ON public.registered_avs_operators USING btree (block_number);


--
-- Name: idx_staker_delegation_changes_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_staker_delegation_changes_block ON public.staker_delegation_changes USING btree (block_number);


--
-- Name: idx_staker_delegation_changes_staker_operator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_staker_delegation_changes_staker_operator ON public.staker_delegation_changes USING btree (staker, operator);


--
-- Name: idx_staker_share_changes_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_staker_share_changes_block ON public.staker_share_changes USING btree (block_number);


--
-- Name: idx_staker_share_changes_staker_strat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_staker_share_changes_staker_strat ON public.staker_share_changes USING btree (staker, strategy);


--
-- Name: idx_staker_shares_block; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_staker_shares_block ON public.staker_shares USING btree (block_number);


--
-- Name: idx_staker_shares_staker_strategy; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_staker_shares_staker_strategy ON public.staker_shares USING btree (staker, strategy);


--
-- Name: idx_transaciton_logs_block_number; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transaciton_logs_block_number ON public.transaction_logs USING btree (block_number);


--
-- Name: idx_transaction_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_transaction_hash ON public.transaction_logs USING btree (transaction_hash, log_index);


--
-- Name: idx_transaction_logs_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transaction_logs_address ON public.transaction_logs USING btree (address);


--
-- Name: idx_transactions_block_number; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transactions_block_number ON public.transactions USING btree (block_number);


--
-- Name: idx_transactions_bytecode_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transactions_bytecode_hash ON public.transactions USING btree (bytecode_hash);


--
-- Name: idx_transactions_from_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transactions_from_address ON public.transactions USING btree (from_address);


--
-- Name: idx_transactions_to_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transactions_to_address ON public.transactions USING btree (to_address);


--
-- Name: idx_uniq_proxy_contract; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_uniq_proxy_contract ON public.proxy_contracts USING btree (block_number, contract_address);


--
-- Name: idx_unique_operator_restaked_strategies; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_unique_operator_restaked_strategies ON public.operator_restaked_strategies USING btree (block_number, operator, avs, strategy);


--
-- Name: transactions_contract_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX transactions_contract_address ON public.transactions USING btree (contract_address);


--
-- Name: transaction_logs fk_transaction_hash_sequence_id_key; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transaction_logs
    ADD CONSTRAINT fk_transaction_hash_sequence_id_key FOREIGN KEY (transaction_hash, block_sequence_id) REFERENCES public.transactions(transaction_hash, block_sequence_id) ON DELETE CASCADE;


--
-- Name: transaction_logs transaction_logs_block_sequence_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transaction_logs
    ADD CONSTRAINT transaction_logs_block_sequence_id_fkey FOREIGN KEY (block_sequence_id) REFERENCES public.blocks(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_block_sequence_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_block_sequence_id_fkey FOREIGN KEY (block_sequence_id) REFERENCES public.blocks(id) ON DELETE CASCADE;


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: -
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- PostgreSQL database dump complete
--

--
-- PostgreSQL database dump
--

-- Dumped from database version 14.11 (Homebrew)
-- Dumped by pg_dump version 15.7 (Homebrew)

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
-- Data for Name: migrations; Type: TABLE DATA; Schema: public; Owner: seanmcgary
--

INSERT INTO public.migrations VALUES ('202405150900_bootstrapDb', '2024-05-17 09:13:15.57261-05', NULL);
INSERT INTO public.migrations VALUES ('202405150917_insertContractAbi', '2024-05-17 09:13:15.575844-05', NULL);
INSERT INTO public.migrations VALUES ('202405151523_addTransactionToFrom', '2024-05-17 09:13:15.578099-05', NULL);
INSERT INTO public.migrations VALUES ('202405170842_addBlockInfoToTransactionLog', '2024-05-17 09:13:15.580354-05', NULL);
INSERT INTO public.migrations VALUES ('202405171056_unverifiedContracts', '2024-05-17 11:00:17.149086-05', NULL);
INSERT INTO public.migrations VALUES ('202405171345_addUpdatedPaymentCoordinatorAbi', '2024-05-17 13:51:24.584807-05', NULL);
INSERT INTO public.migrations VALUES ('202405201503_fixTransactionHashConstraint', '2024-05-20 15:10:45.476856-05', NULL);
INSERT INTO public.migrations VALUES ('202405300925_addUniqueBlockConstraint', '2024-05-30 09:33:13.115195-05', NULL);
INSERT INTO public.migrations VALUES ('202405312008_indexTransactionContractAddress', '2024-05-31 21:13:37.099393-05', NULL);
INSERT INTO public.migrations VALUES ('202405312134_handleProxyContracts', '2024-05-31 22:21:46.84577-05', NULL);
INSERT INTO public.migrations VALUES ('202406030920_addCheckedForProxyFlag', '2024-06-03 10:09:02.176827-05', NULL);
INSERT INTO public.migrations VALUES ('202406031946_addSerialIdToContracts', '2024-06-04 08:54:09.723152-05', NULL);
INSERT INTO public.migrations VALUES ('202406051937_addBytecodeIndex', '2024-06-05 19:54:36.665099-05', NULL);
INSERT INTO public.migrations VALUES ('202406071318_indexTransactionLogBlockNumber', '2024-06-07 13:20:19.291429-05', NULL);
INSERT INTO public.migrations VALUES ('202406110848_transactionLogsContractIndex', '2024-06-11 11:26:43.316616-05', NULL);
INSERT INTO public.migrations VALUES ('202406141007_addCheckedForAbiFlag', '2024-06-14 10:13:35.067238-05', NULL);
INSERT INTO public.migrations VALUES ('202406251424_addTransactionLogsOutputDataColumn', '2024-06-25 14:29:47.494612-05', NULL);
INSERT INTO public.migrations VALUES ('202406251426_addTransactionIndexes', '2024-06-25 14:29:47.500543-05', NULL);
INSERT INTO public.migrations VALUES ('202407101440_addOperatorRestakedStrategiesTable', '2024-07-11 09:48:48.933519-05', NULL);
INSERT INTO public.migrations VALUES ('202407110946_addBlockTimeToRestakedStrategies', '2024-07-11 09:49:17.325774-05', NULL);
INSERT INTO public.migrations VALUES ('202407111116_addAvsDirectoryAddress', '2024-08-20 16:12:38.890285-05', NULL);
INSERT INTO public.migrations VALUES ('202407121407_updateProxyContractIndex', '2024-08-20 16:12:38.893514-05', NULL);
INSERT INTO public.migrations VALUES ('202408200934_eigenlayerStateTables', '2024-08-20 16:12:38.950484-05', NULL);


--
-- PostgreSQL database dump complete
--

