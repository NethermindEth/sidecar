#ifndef CALCULATIONS_H
#define CALCULATIONS_H

#include <sqlite3ext.h>

int init_python();
void finalize_python();
char* call_python_func(const char* func_name, const char* arg1, const char* arg2);

char* _pre_nile_tokens_per_day(const char* tokens);
void pre_nile_tokens_per_day(sqlite3_context *context, int argc, sqlite3_value **argv);

char* _amazon_staker_token_rewards(const char* sp, const char* tpd);
void amazon_staker_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv);

char* _nile_staker_token_rewards(const char* sp, const char* tpd);
void nile_staker_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv);

char* _staker_token_rewards(const char* sp, const char* tpd);
void staker_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv);

char* _amazon_operator_token_rewards(const char* totalStakerOperatorTokens);
void amazon_operator_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv);

char* _nile_operator_token_rewards(const char* totalStakerOperatorTokens);
void nile_operator_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv);

#endif // CALCULATIONS_H
