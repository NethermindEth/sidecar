
## mainnet reduced staker delegation transactions

```sql
select
    transaction_hash,
    transaction_index,
    block_number,
    address,
    arguments,
    event_name,
    log_index,
    output_data
from transaction_logs
where
    block_number < 20613003
    and address = '0x39053d51b77dc0d36036fc1fcc8cb819df8ef37a'
    and event_name in ('StakerUndelegated', 'StakerDelegated')
```

## expected results

```sql
select
    count(*)
from dbt_mainnet_ethereum_rewards.staker_delegation_status
where block_number < 20613003
```
