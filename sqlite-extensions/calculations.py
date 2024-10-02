from decimal import Decimal, getcontext, ROUND_HALF_UP, ROUND_UP, ROUND_DOWN

def preNileTokensPerDay(tokens: str) -> str:
    big_amount = float(tokens)
    div = 0.999999999999999
    res = big_amount * div

    res_str = "{}".format(res)
    return "{}".format(int(Decimal(res_str)))

def amazonStakerTokenRewards(sp:str, tpd:str) -> str:
    getcontext().prec = 15
    stakerProportion = Decimal(sp)
    tokensPerDay = Decimal(tpd)

    decimal_res = Decimal(stakerProportion * tokensPerDay)

    getcontext().prec = 20
    rounded = decimal_res.quantize(Decimal('1'), rounding=ROUND_HALF_UP)

    return "{}".format(rounded)


def nileStakerTokenRewards(sp:str, tpd:str) -> str:
    stakerProportion = Decimal(sp)
    tokensPerDay = Decimal(tpd)

    decimal_res = Decimal(stakerProportion * tokensPerDay)
    # Truncate to 0.x decimals
    truncated = decimal_res.quantize(Decimal('0.1'), rounding=ROUND_UP)

    # Bankers rounding to match postgres
    rounded = truncated.quantize(Decimal('1'), rounding=ROUND_HALF_UP)

    return "{}".format(rounded)

def stakerTokenRewards(sp:str, tpd:str) -> str:
    stakerProportion = float(sp)
    tokensPerDay = int(tpd)

    decimal_res = stakerProportion * tokensPerDay

    parts = str(decimal_res).split("+")
    # Need more precision
    if len(parts) == 2 and int(parts[1]) > 16:
        stakerProportion = Decimal(sp)
        tokensPerDay = Decimal(tpd)

        getcontext().prec = 17
        getcontext().rounding = ROUND_DOWN
        decimal_res = stakerProportion * tokensPerDay

        return "{}".format(int(decimal_res))

    return "{}".format(int(decimal_res))


def amazonOperatorTokenRewards(totalStakerOperatorTokens:str) -> str:
    totalStakerOperatorTokens = Decimal(totalStakerOperatorTokens)

    operatorTokens = totalStakerOperatorTokens * Decimal(0.1)

    rounded = operatorTokens.quantize(Decimal('1'), rounding=ROUND_HALF_UP)

    return "{}".format(rounded)

def nileOperatorTokenRewards(totalStakerOperatorTokens:str) -> str:
    if totalStakerOperatorTokens[-1] == "0":
        return "{}".format(int(totalStakerOperatorTokens) // 10)
    totalStakerOperatorTokens = Decimal(totalStakerOperatorTokens)
    operatorTokens = Decimal(str(totalStakerOperatorTokens)) * Decimal(0.1)
    rounded = operatorTokens.quantize(Decimal('1'), rounding=ROUND_HALF_UP)
    return "{}".format(rounded)

def bigGt(a:str, b:str) -> bool:
    return Decimal(a) > Decimal(b)
