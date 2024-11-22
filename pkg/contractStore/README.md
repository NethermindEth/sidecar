## Generating contracts from BlockLake

```sql
-- base query
with all_contracts as (
    select
        contract_address,
        proxy_contract_address,
        block_number
    from proxy_contracts
    where contract_address in (
        -- core contracts for network (e.g. mainnet, testnet, preprod)
        '0xacc1fb458a1317e886db376fc8141540537e68fe',
        '0x30770d7e3e71112d7a6b7259542d1f680a70e315',
        '0xdfb5f6ce42aaa7830e94ecfccad411bef4d4d5b6',
        '0xa44151489861fe9e3055d95adc98fbd462b948e7',
        '0x055733000064333caddbc92763c58bf0192ffebf'
    )
),
distinct_contracts as (
    select distinct contract_address from all_contracts
    union all
    select distinct proxy_contract_address from all_contracts
)


-- proxy contracts
select
    contract_address,
    contract_abi,
    bytecode_hash
from contracts
where contract_address in (select contract_address from distinct_contracts)

-- implementation contracts
select * from all_contracts
```
