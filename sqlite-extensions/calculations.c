#define PY_SSIZE_T_CLEAN
#include <Python.h>
#include <stdlib.h>
#include <stdio.h>
#include "calculations.h"
SQLITE_EXTENSION_INIT1

static PyObject *pModule = NULL;
static int python_initialized = 0;

int ensure_python_initialized() {
    if (!python_initialized) {
        Py_Initialize();
        // As of Python 3.7, the GIL is created automatically,
        // and threads are initialized by default
        python_initialized = 1;
    }
    return python_initialized;
}

void finalize_python() {
    // Reacquire the GIL before finalizing
    PyGILState_STATE gstate = PyGILState_Ensure();
    Py_XDECREF(pModule);
    Py_Finalize();
}

char* call_python_func(const char* func_name, const char* arg1, const char* arg2) {
    const char *module_name = "calculations";
    char *result = NULL;
    PyGILState_STATE gstate;
    PyObject *pName, *pModule, *pFunc, *pArgs, *pValue;

    if (!ensure_python_initialized()) {
        return NULL;
    }

    gstate = PyGILState_Ensure();

    pName = PyUnicode_DecodeFSDefault(module_name);
    pModule = PyImport_Import(pName);
    Py_DECREF(pName);

    if (pModule != NULL) {
        pFunc = PyObject_GetAttrString(pModule, func_name);

        if (pFunc && PyCallable_Check(pFunc)) {
            if (arg2 == NULL) {
                pArgs = PyTuple_Pack(1, PyUnicode_DecodeFSDefault(arg1));
            } else {
                pArgs = PyTuple_Pack(2, PyUnicode_DecodeFSDefault(arg1), PyUnicode_DecodeFSDefault(arg2));
            }

            pValue = PyObject_CallObject(pFunc, pArgs);
            Py_DECREF(pArgs);

            if (pValue != NULL) {
                PyObject *str_obj = PyObject_Str(pValue);
                const char *str_result = PyUnicode_AsUTF8(str_obj);
                result = strdup(str_result);
                Py_DECREF(str_obj);
                Py_DECREF(pValue);
            }
        }
        Py_XDECREF(pFunc);
        Py_DECREF(pModule);
    }

    if (PyErr_Occurred()) {
        PyErr_Print();
    }

    PyGILState_Release(gstate);
    return result;
}
int call_bool_python_func(const char* func_name, const char* arg1, const char* arg2) {
    const char *module_name = "calculations";
    int result = 0;
    PyGILState_STATE gstate;
    PyObject *pName, *pModule, *pFunc, *pArgs, *pValue;

    if (!ensure_python_initialized()) {
        return 0;
    }

    gstate = PyGILState_Ensure();

    pName = PyUnicode_DecodeFSDefault(module_name);
    pModule = PyImport_Import(pName);
    Py_DECREF(pName);

    if (pModule != NULL) {
        pFunc = PyObject_GetAttrString(pModule, func_name);

        if (pFunc && PyCallable_Check(pFunc)) {
            if (arg2 == NULL) {
                pArgs = PyTuple_Pack(1, PyUnicode_DecodeFSDefault(arg1));
            } else {
                pArgs = PyTuple_Pack(2, PyUnicode_DecodeFSDefault(arg1), PyUnicode_DecodeFSDefault(arg2));
            }

            pValue = PyObject_CallObject(pFunc, pArgs);
            Py_DECREF(pArgs);

            if (pValue != NULL) {
                if (PyObject_IsTrue(pValue)) {
                    result = 1;
                } else {
                    result = 0;
                }
                Py_DECREF(pValue);
            }
        }
        Py_XDECREF(pFunc);
        Py_DECREF(pModule);
    }

    if (PyErr_Occurred()) {
        PyErr_Print();
    }

    PyGILState_Release(gstate);
    return result;
}

// _pre_nile_tokens_per_day is a non-sqlite3 function that is called by pre_nile_tokens_per_day
char* _pre_nile_tokens_per_day(const char* tokens) {
    return call_python_func("preNileTokensPerDay", tokens, NULL);
}

// Your custom function
void pre_nile_tokens_per_day(sqlite3_context *context, int argc, sqlite3_value **argv) {
    if (argc != 1) {
        sqlite3_result_error(context, "pre_nile_tokens_per_day() requires exactly one argument", -1);
        return;
    }
    const char* input = (const char*)sqlite3_value_text(argv[0]);
    if (!input) {
        sqlite3_result_null(context);
        return;
    }

    char* tokens = _pre_nile_tokens_per_day(input);
    if (!tokens) {
        sqlite3_result_null(context);
        return;
    }

    sqlite3_result_text(context, tokens, -1, SQLITE_TRANSIENT);
}

char* _amazon_staker_token_rewards(const char* sp, const char* tpd) {
    return call_python_func("amazonStakerTokenRewards", sp, tpd);
}

void amazon_staker_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv) {
    if (argc != 2) {
        sqlite3_result_error(context, "amazon_staker_token_rewards() requires two arguments", -1);
        return;
    }
    const char* sp = (const char*)sqlite3_value_text(argv[0]);
    if (!sp) {
        sqlite3_result_null(context);
        return;
    }

    const char* tpd = (const char*)sqlite3_value_text(argv[1]);
    if (!tpd) {
        sqlite3_result_null(context);
        return;
    }

    char* tokens = _amazon_staker_token_rewards(sp, tpd);
    if (!tokens) {
        sqlite3_result_null(context);
        return;
    }

    sqlite3_result_text(context, tokens, -1, SQLITE_TRANSIENT);
}

char* _nile_staker_token_rewards(const char* sp, const char* tpd) {
    return call_python_func("nileStakerTokenRewards", sp, tpd);
}

void nile_staker_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv) {
    if (argc != 2) {
        sqlite3_result_error(context, "nile_staker_token_rewards() requires two arguments", -1);
        return;
    }
    const char* sp = (const char*)sqlite3_value_text(argv[0]);
    if (!sp) {
        sqlite3_result_null(context);
        return;
    }

    const char* tpd = (const char*)sqlite3_value_text(argv[1]);
    if (!tpd) {
        sqlite3_result_null(context);
        return;
    }

    char* tokens = _nile_staker_token_rewards(sp, tpd);
    if (!tokens) {
        sqlite3_result_null(context);
        return;
    }

    sqlite3_result_text(context, tokens, -1, SQLITE_TRANSIENT);
}

char* _staker_token_rewards(const char* sp, const char* tpd) {
    return call_python_func("stakerTokenRewards", sp, tpd);
}

void staker_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv) {
    if (argc != 2) {
        sqlite3_result_error(context, "staker_token_rewards() requires two arguments", -1);
        return;
    }
    const char* sp = (const char*)sqlite3_value_text(argv[0]);
    if (!sp) {
        sqlite3_result_null(context);
        return;
    }

    const char* tpd = (const char*)sqlite3_value_text(argv[1]);
    if (!tpd) {
        sqlite3_result_null(context);
        return;
    }

    char* tokens = _staker_token_rewards(sp, tpd);
    if (!tokens) {
        sqlite3_result_null(context);
        return;
    }

    sqlite3_result_text(context, tokens, -1, SQLITE_TRANSIENT);
}

char* _amazon_operator_token_rewards(const char* totalStakerOperatorTokens) {
    return call_python_func("amazonOperatorTokenRewards", totalStakerOperatorTokens, NULL);
}

void amazon_operator_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv) {
    if (argc != 1) {
        sqlite3_result_error(context, "amazon_operator_token_rewards() requires exactly one argument", -1);
        return;
    }
    const char* input = (const char*)sqlite3_value_text(argv[0]);
    if (!input) {
        sqlite3_result_null(context);
        return;
    }

    char* tokens = _amazon_operator_token_rewards(input);
    if (!tokens) {
        sqlite3_result_null(context);
        return;
    }

    sqlite3_result_text(context, tokens, -1, SQLITE_TRANSIENT);
}


char* _nile_operator_token_rewards(const char* totalStakerOperatorTokens) {
    return call_python_func("nileOperatorTokenRewards", totalStakerOperatorTokens, NULL);
}
void nile_operator_token_rewards(sqlite3_context *context, int argc, sqlite3_value **argv) {
    if (argc != 1) {
        sqlite3_result_error(context, "nile_operator_token_rewards() requires exactly one argument", -1);
        return;
    }
    const char* input = (const char*)sqlite3_value_text(argv[0]);
    if (!input) {
        sqlite3_result_null(context);
        return;
    }

    char* tokens = _nile_operator_token_rewards(input);
    if (!tokens) {
        sqlite3_result_null(context);
        return;
    }

    sqlite3_result_text(context, tokens, -1, SQLITE_TRANSIENT);
}

int _big_gt(const char* a, const char* b) {
    return call_bool_python_func("bigGt", a, b);
}
void big_gt(sqlite3_context *context, int argc, sqlite3_value **argv){
    if (argc != 2) {
        sqlite3_result_error(context, "big_gt() requires exactly two arguments", -1);
        return;
    }
    const char* sp = (const char*)sqlite3_value_text(argv[0]);
    if (!sp) {
        sqlite3_result_null(context);
        return;
    }

    const char* tpd = (const char*)sqlite3_value_text(argv[1]);
    if (!tpd) {
        sqlite3_result_null(context);
        return;
    }

    int is_greater = _big_gt(sp, tpd);

    sqlite3_result_int(context, is_greater ? 1 : 0);
}

int sqlite3_calculations_init(sqlite3 *db, char **pzErrMsg, const sqlite3_api_routines *pApi) {
    SQLITE_EXTENSION_INIT2(pApi);

    int rc;
    rc = sqlite3_create_function(db, "pre_nile_tokens_per_day", 1, SQLITE_UTF8 | SQLITE_DETERMINISTIC, 0, pre_nile_tokens_per_day, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }

    rc = sqlite3_create_function(db, "amazon_staker_token_rewards", 2, SQLITE_UTF8 | SQLITE_DETERMINISTIC, 0, amazon_staker_token_rewards, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }

    rc = sqlite3_create_function(db, "nile_staker_token_rewards", 2, SQLITE_UTF8 | SQLITE_DETERMINISTIC, 0, nile_staker_token_rewards, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }

    rc = sqlite3_create_function(db, "staker_token_rewards", 2, SQLITE_UTF8 | SQLITE_DETERMINISTIC, 0, staker_token_rewards, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }

    rc = sqlite3_create_function(db, "amazon_operator_token_rewards", 1, SQLITE_UTF8 | SQLITE_DETERMINISTIC, 0, amazon_operator_token_rewards, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }

    rc = sqlite3_create_function(db, "nile_operator_token_rewards", 1, SQLITE_UTF8 | SQLITE_DETERMINISTIC, 0, nile_operator_token_rewards, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }

    rc = sqlite3_create_function(db, "big_gt", 2, SQLITE_DETERMINISTIC, 0, big_gt, 0, 0);
    if (rc != SQLITE_OK) {
        fprintf(stderr, "Failed to create function: %s\n", sqlite3_errmsg(db));
        return rc;
    }
    return SQLITE_OK;
}

void sqlite3_calculations_shutdown(void) {
    if (python_initialized) {
        PyGILState_Ensure();
        Py_Finalize();
        python_initialized = 0;
    }
}
