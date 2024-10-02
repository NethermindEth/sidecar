CC = gcc
AR = ar
RANLIB = ranlib
INSTALL_NAME_TOOL = install_name_tool

PYTHON_CONFIG = python3-config
PYTHON_VERSION = $(shell python3 -c "import sys; print('{}.{}'.format(sys.version_info.major, sys.version_info.minor))")
PYTHON_LIBDIR := $(shell $(PYTHON_CONFIG) --prefix)/lib

# Project structure
SRC_DIR = .
BUILD_DIR = build
LIB_DIR = $(BUILD_DIR)/lib
OBJ_DIR = $(BUILD_DIR)/obj
SRC_DIR_ABS=$(shell cd $(SRC_DIR); pwd)

# Ensure build directories exist
$(shell mkdir -p $(LIB_DIR) $(OBJ_DIR))

# Base flags
CFLAGS = -g -fPIC
LDFLAGS =

INCLUDE_DIRS = $(SRC_DIR)
CFLAGS += $(foreach dir,$(INCLUDE_DIRS),-I$(dir))

# Python flags
PYTHON_CFLAGS := $(shell $(PYTHON_CONFIG) --includes)
PYTHON_LDFLAGS := $(shell $(PYTHON_CONFIG) --ldflags)

# SQLite flags (adjust the path as necessary)
SQLITE_DIR = /opt/homebrew/opt/sqlite
CFLAGS += -I$(SQLITE_DIR)/include
LDFLAGS += -L$(SQLITE_DIR)/lib -lsqlite3

CFLAGS += $(PYTHON_CFLAGS)
LDFLAGS += $(PYTHON_LDFLAGS) -L$(PYTHON_LIBDIR) -lpython$(PYTHON_VERSION)

# Source files
SOURCES = $(wildcard $(SRC_DIR)/*.c)
OBJECTS = $(patsubst $(SRC_DIR)/%.c,$(OBJ_DIR)/%.o,$(SOURCES))

# Shared library target
SHARED_LIB = $(LIB_DIR)/libcalculations.dylib

# Static library target
STATIC_LIB = $(LIB_DIR)/libcalculations.a

# Shared library rule
$(SHARED_LIB): $(OBJECTS)
	$(CC) -shared $(CFLAGS) -o $@ $^ $(LDFLAGS)
	$(INSTALL_NAME_TOOL) -id $(SRC_DIR_ABS)/$(SHARED_LIB) $@

# Static library rule
$(STATIC_LIB): $(OBJECTS)
	$(AR) rcs $@ $^
	$(RANLIB) $@

# Object file rule
$(OBJ_DIR)/%.o: $(SRC_DIR)/%.c
	$(CC) $(CFLAGS) -c $< -o $@

# Phony targets
.PHONY: all shared static clean

all: shared static

shared: $(SHARED_LIB)

static: $(STATIC_LIB)

clean:
	rm -rf $(BUILD_DIR)

# Backwards compatibility
.PHONY: sqlite-extensions
sqlite-extensions: shared
