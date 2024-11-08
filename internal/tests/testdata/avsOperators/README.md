
## mainnet reduced transactions

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
    address = '0x135dda560e946695d6f155dacafc6f1f25c1f5af'
    and event_name = 'OperatorAVSRegistrationStatusUpdated'
    and block_number < 20613003
``
