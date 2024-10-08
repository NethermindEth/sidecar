### Running

```bash
PROJECT_PATH=$(pwd)
PYTHONPATH="${PROJECT_PATH}/sqlite-extensions:$PYTHONPATH" lldb -- /opt/homebrew/opt/sqlite/bin/sqlite3
```

### Go test example
```bash
PROJECT_PATH=$(pwd)
PYTHONPATH="${PROJECT_PATH}/sqlite-extensions:$PYTHONPATH" CGO_ENABLED=1 TESTING=true go test -v ./internal/types/numbers -v -p 1 -run '^Test_Numbers$'
```
