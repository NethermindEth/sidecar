

### Mainnet reduced

```sql
select * from transaction_logs
where
    block_number <= 20470136
    and (
        -- delegation manager
        (address = '0x39053d51b77dc0d36036fc1fcc8cb819df8ef37a' and (event_name = 'WithdrawalQueued' or event_name = 'WithdrawalMigrated'))
        -- strategy manager
        or (address = '0x858646372cc42e1a627fce94aa7a7033e7cf075a' and (event_name = 'ShareWithdrawalQueued' or event_name = 'Deposit'))
        -- eigenpod shares
        or (address = '0x91e677b07f7af907ec9a428aafa9fc14a0d3a338' and event_name = 'PodSharesUpdated')
    )
order by block_number asc, log_index asc

```
