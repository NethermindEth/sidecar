package _202411041043_blockNumberFkConstraint

import (
	"database/sql"
	"fmt"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table delegated_stakers add constraint delegated_stakers_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table combined_rewards add constraint combined_rewards_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table operator_restaked_strategies add constraint operator_restaked_strategies_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table operator_share_deltas add constraint operator_share_deltas_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table operator_shares add constraint operator_shares_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table registered_avs_operators add constraint registered_avs_operators_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table reward_submissions add constraint reward_submissions_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table staker_delegation_changes add constraint staker_delegation_changes_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table staker_shares add constraint staker_shares_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table state_roots add constraint state_roots_block_number_fkey foreign key (eth_block_number) references blocks (number) on delete cascade`,
		`alter table submitted_distribution_roots add constraint submitted_distribution_roots_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table transaction_logs add constraint transaction_logs_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table transactions add constraint transactions_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			fmt.Printf("Failed to run migration query: %s - %+v\n", query, err)
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202411041043_blockNumberFkConstraint"
}
