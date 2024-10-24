package numbers

// CalculateAmazonStakerTokenRewards calculates the Amazon token rewards for a given staker proportion and tokens per day
// cast(staker_proportion * tokens_per_day AS DECIMAL(38,0))
func CalculateAmazonStakerTokenRewards(stakerProportion string, tokensPerDay string) (string, error) {
	panic("Implements me")
}

// CalculateNileStakerTokenRewards calculates the tokens to be rewarded for a given staker proportion and tokens per day
// (staker_proportion * tokens_per_day)::text::decimal(38,0)
func CalculateNileStakerTokenRewards(stakerProportion string, tokensPerDay string) (string, error) {
	panic("Implements me")
}

// CalculatePostNileStakerTokenRewards calculates the tokens to be rewarded for a given staker proportion and tokens per day
// FLOOR(staker_proportion * tokens_per_day_decimal)
func CalculatePostNileStakerTokenRewards(stakerProportion string, tokensPerDay string) (string, error) {
	panic("Implements me")
}

// CalculateAmazonOperatorTokens calculates the operator payout portion for rewards (10% of total)
//
// cast(total_staker_operator_payout * 0.10 AS DECIMAL(38,0))
func CalculateAmazonOperatorTokens(totalStakerPayout string) (string, error) {
	panic("Implements me")
}

// CalculateNileOperatorTokens calculates the operator payout portion for rewards (10% of total)
//
// (total_staker_operator_payout * 0.10)::text::decimal(38,0)
func CalculateNileOperatorTokens(totalStakerPayout string) (string, error) {
	panic("Implements me")
}

// CalculatePostNileOperatorTokens calculates the operator payout portion for rewards (10% of total)
//
// floor(total_staker_operator_payout * 0.10)
func CalculatePostNileOperatorTokens(totalStakerPayout string) (string, error) {
	panic("Implements me")
}

// PreNileTokensPerDay calculates the tokens per day for pre-nile rewards, rounded to 15 sigfigs
//
// Not gonna lie, this is pretty annoying that it has to be this way, but in order to support backwards compatibility
// with the current/old rewards system where postgres was lossy, we have to do this.
func PreNileTokensPerDay(tokensPerDay string) (string, error) {
	panic("Implements me")
}

func BigGt(a, b string) (bool, error) {
	panic("Implements me")
}
