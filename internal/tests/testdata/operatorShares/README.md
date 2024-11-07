

### Mainnet reduced deltas

```sql
SELECT
    operator,
    strategy,
    shares,
    transaction_hash,
    log_index,
    block_time,
    block_date,
    block_number
FROM (
         SELECT operator, strategy, shares, transaction_hash, log_index, block_time, block_date, block_number
         FROM dbt_mainnet_ethereum_rewards.operator_share_increases
         where block_date < '2024-08-20'

         UNION ALL

         SELECT operator, strategy, shares * -1 AS shares, transaction_hash, log_index, block_time, block_date, block_number
         FROM dbt_mainnet_ethereum_rewards.operator_share_decreases
         where block_date < '2024-08-20'
     ) combined_shares

```
