# Generate configuration files with correct absolute paths.

ROOT?=$(shell git root)
KORREL8R_CONFIG_DIR?=$(ROOT)/etc/korrel8r
CLAUDE_CONFIG=$(HOME)/.config/Claude/claude_desktop_config.json

all: claude

mcpconfig.json: mcpconfig-envsubst.json
	KORREL8R_CONFIG_DIR=$(KORREL8R_CONFIG_DIR) envsubst < $< > $@

claude-config: $(CLAUDE_CONFIG)
$(CLAUDE_CONFIG): mcpconfig.json
	cp -f $< $@

clean:
	rm -f mcpconfig.json $(CLAUDE_CONFIG)
