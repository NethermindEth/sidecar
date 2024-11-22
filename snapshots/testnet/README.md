```sql
select
    b.number,
    b.hash,
    coalesce(p.hash, '0x0000000000000000000000000000000000000000') as parent_hash,
    b.block_time
from blocks as b
         left join blocks as p on (p.number = b.number - 1)
where b.number <= 2712736;

-- transactions
select
    block_number,
    transaction_hash,
    transaction_index,
    from_address,
    to_address,
    contract_address,
    bytecode_hash,
    gas_used,
    cumulative_gas_used,
    effective_gas_price
from transactions
where
    block_number <= 2712736
  and transaction_hash in (
    select distinct(transaction_hash) from transaction_logs where address in ('0xacc1fb458a1317e886db376fc8141540537e68fe', '0x30770d7e3e71112d7a6b7259542d1f680a70e315', '0xdfb5f6ce42aaa7830e94ecfccad411bef4d4d5b6', '0xa44151489861fe9e3055d95adc98fbd462b948e7', '0x055733000064333caddbc92763c58bf0192ffebf')
)

-- transaction logs
select
    transaction_hash,
    transaction_index,
    block_number,
    address,
    arguments,
    event_name,
    log_index,
    output_data
from transaction_logs where
    address in ('0xacc1fb458a1317e886db376fc8141540537e68fe', '0x30770d7e3e71112d7a6b7259542d1f680a70e315', '0xdfb5f6ce42aaa7830e94ecfccad411bef4d4d5b6', '0xa44151489861fe9e3055d95adc98fbd462b948e7', '0x055733000064333caddbc92763c58bf0192ffebf')
                        and block_number <= 2712736


-- operator restaked strategies
select
    block_number,
    operator,
    avs,
    strategy,
    block_time,
    avs_directory_address
from operator_restaked_strategies
where
    avs_directory_address = '0x055733000064333caddbc92763c58bf0192ffebf'
  and block_number > 2321960
  and block_number <= 2712736

-- avs operators
select
    operator,
    avs,
    block_number,
    log_index,
    case when status = 1 then true else false end as registered
from dbt_testnet_holesky_rewards.operator_avs_registrations
where
    block_number <= 2712736

-- disabled distribution roots
select
    arguments #>> '{0,Value}' as root_index,
    block_number,
    log_index,
    transaction_hash
from transaction_logs
where
    address = '0xacc1fb458a1317e886db376fc8141540537e68fe'
  and event_name = 'DistributionRootDisabled'

-- operator shares

with share_increases as (
    SELECT
    lower(t.arguments #>> '{0,Value}') as operator,
    lower(t.output_data ->> 'strategy') as strategy,
    lower(t.output_data ->> 'staker') as staker,
    (t.output_data ->> 'shares')::numeric(78,0) as shares,
    t.transaction_hash,
    t.log_index,
    b.block_time,
    to_char(b.block_time, 'YYYY-MM-DD') AS block_date,
    t.block_number
    FROM transaction_logs t
    LEFT JOIN blocks b ON t.block_sequence_id = b.id
    WHERE t.address = '0xa44151489861fe9e3055d95adc98fbd462b948e7' -- delegation manager
    AND t.event_name = 'OperatorSharesIncreased'
    AND block_number <= 2712736
    ),
    share_decreases as (
    SELECT
    lower(t.arguments #>> '{0,Value}') as operator,
    lower(t.output_data ->> 'strategy') as strategy,
    lower(t.output_data ->> 'staker') as staker,
    (t.output_data ->> 'shares')::numeric(78,0) as shares,
    t.transaction_hash,
    t.log_index,
    b.block_time,
    to_char(b.block_time, 'YYYY-MM-DD') AS block_date,
    t.block_number
    FROM transaction_logs t
    LEFT JOIN blocks b ON t.block_sequence_id = b.id
    WHERE t.address = '0xa44151489861fe9e3055d95adc98fbd462b948e7' -- delegation manager
    AND t.event_name = 'OperatorSharesDecreased'
    AND block_number <= 2712736
    ),
    union_shares as (
    SELECT operator, staker, strategy, shares, transaction_hash, log_index, block_time, block_date, block_number
    FROM share_increases

    UNION ALL

    SELECT operator, staker, strategy, shares * -1 AS shares, transaction_hash, log_index, block_time, block_date, block_number
    FROM share_decreases
    )
select
    operator,
    staker,
    strategy,
    shares,
    transaction_hash,
    log_index,
    block_number,
    block_time,
    block_date
FROM union_shares


-- reward submissions
select
    avs,
    reward_hash,
    token,
    amount,
    strategy,
    strategy_index,
    multiplier,
    start_timestamp,
    end_timestamp,
    duration,
    block_number,
    reward_type,
    transaction_hash,
    log_index
from dbt_testnet_holesky_rewards.rewards_combined
where block_number <= 2712736


-- staker delegations
SELECT
    staker,
    operator,
    block_number,
    CASE when src = 'undelegations' THEN false ELSE true END AS delegated,
    transaction_hash,
    log_index
FROM (
         SELECT *, 'undelegations' AS src FROM dbt_testnet_holesky_rewards.staker_undelegations
         UNION ALL
         SELECT *, 'delegations' AS src FROM dbt_testnet_holesky_rewards.staker_delegations
     ) as delegations_combined
where block_number <= 2712736

-- staker shares

SELECT
    staker,
    strategy,
    shares,
    transaction_hash,
    log_index,
    strategy_index,
    block_time,
    block_date,
    block_number
FROM (
         SELECT staker, strategy, shares, 0 as strategy_index, transaction_hash, log_index, block_time, block_date, block_number
         FROM dbt_testnet_holesky_rewards.staker_deposits

         UNION ALL

         -- Subtract m1 & m2 withdrawals
         SELECT staker, strategy, shares * -1, 0 as strategy_index, transaction_hash, log_index, block_time, block_date, block_number
         FROM dbt_testnet_holesky_rewards.m1_staker_withdrawals

         UNION ALL

         SELECT staker, strategy, shares * -1, strategy_index, transaction_hash, log_index, block_time, block_date, block_number
         FROM dbt_testnet_holesky_rewards.m2_staker_withdrawals

         UNION all

         -- Shares in eigenpod are positive or negative, so no need to multiply by -1
         SELECT staker, '0xbeac0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeebeac0' as strategy, shares, 0 as strategy_index, transaction_hash, log_index, block_time, block_date, block_number
         FROM dbt_testnet_holesky_rewards.eigenpod_shares
     ) combined_staker_shares
where block_number <= 2712736

-- submitted distribution roots

    with raw_roots as (
select
    tl.arguments #>> '{{0,Value}}' as root_index,
    tl.arguments #>> '{{1,Value}}' as root,
    cast(tl.arguments #>> '{{2,Value}}'as bigint) AS rewards_calculation_end,
    'snapshot' as rewards_calculation_end_unit,
    CAST(tl.output_data ->> 'activatedAt' AS bigint) AS activated_at,
    'timestamp' as activated_at_unit,
    tl.transaction_hash,
    tl.block_number,
    log_index
from transaction_logs as tl
where
    address = '0xacc1fb458a1317e886db376fc8141540537e68fe'
    and event_name = 'DistributionRootSubmitted'
    and block_number <= 2712736
)
select
    root_index,
    root,
    to_timestamp(rewards_calculation_end) as rewards_calculation_end,
    rewards_calculation_end_unit,
    to_timestamp(activated_at) as activated_at,
    activated_at_unit,
    transaction_hash,
    block_number as created_at_block_number,
    block_number,
    log_index
from raw_roots;

-- gold
-- select * from dbt_testnet_holesky_rewards.gold_table where snapshot < DATE '2024-09-12' and reward_hash in (select reward_hash from dbt_testnet_holesky_rewards.rewards_combined where block_number <= 2712736)

```
