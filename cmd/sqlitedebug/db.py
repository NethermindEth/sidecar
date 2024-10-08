import sqlite3

conn = sqlite3.connect(":memory:")
conn.enable_load_extension(True)
conn.load_extension("/Users/seanmcgary/Code/sidecar/sqlite-extensions/yolo.dylib")
conn.execute("SELECT my_custom_function('foo')")
conn.close()
