package types

type RewardsClaimed struct {
	Root            string
	Earner          string
	Claimer         string
	Recipient       string
	Token           string
	ClaimedAmount   string
	TransactionHash string
	BlockNumber     uint64
	LogIndex        uint64
}

func (*RewardsClaimed) TableName() string {
	return "rewards_claimed"
}
