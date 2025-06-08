.PHONY: build run clean run install uninstall install-templates info logs


APP_NAME := switchboard-web-ui

BINARY_NAME := $(APP_NAME)
INSTALL_DIR := /usr/local/bin
SERVICE_FILE := $(APP_NAME).service
SYSTEMD_DIR := /etc/systemd/system

USR_SHARE := /usr/share/$(APP_NAME)

# Web templates, e.g. index.html
TEMPLATE_SRC_DIR := templates
TEMPLATE_DST_DIR := $(USR_SHARE)/templates

# Static files like favicon.ico
STATIC_SRC_DIR := static
STATIC_DST_DIR := $(USR_SHARE)/static

VAR_LIB := /var/lib/$(APP_NAME)

# Tokens for access to the site
TOKEN_DIR := $(VAR_LIB)/token

# The 'widgets' to be included in the UI
WIDGET_DIR := $(VAR_LIB)/widget

# The current state, shared with other apps
STATE_DIR := /var/lib/switchboard


run: build
	./$(BINARY_NAME) -config ./config.json


install: build
	sudo install -m 755 $(BINARY_NAME) $(INSTALL_DIR)/
	sudo install -d $(TEMPLATE_DST_DIR)
	sudo install -m 644 $(TEMPLATE_SRC_DIR)/* $(TEMPLATE_DST_DIR)
	sudo install -d $(STATIC_DST_DIR)
	sudo install -m 644 $(STATIC_SRC_DIR)/* $(STATIC_DST_DIR)
	sudo install -d $(TOKEN_DIR)
	sudo install -d $(WIDGET_DIR)
	sudo install -d $(STATE_DIR)
	sudo install -m 644 $(SERVICE_FILE) $(SYSTEMD_DIR)/
	sudo systemctl daemon-reexec
	sudo systemctl enable --now $(BINARY_NAME).service
	@echo "Installed and started $(BINARY_NAME).service"


info:
	systemctl status $(SERVICE_FILE)
	
	
logs:
        journalctl -u $(SERVICE_FILE) -n 20


uninstall:
	sudo systemctl disable --now $(APP_NAME).service
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	sudo rm -rf $(TEMPLATE_DST_DIR)
	sudo rm -rf $(STATIC_DST_DIR)
	sudo rm -rf $(TOKEN_DIR)
	sudo rm -rf $(WIDGET_DIR)
	sudo rm -rf $(STATE_DIR)
	sudo rm -f $(SYSTEMD_DIR)/$(SERVICE_FILE)
	sudo systemctl daemon-reexec
	@echo "Uninstalled $(BINARY_NAME)"
	

build:
	go build -o $(BINARY_NAME) main.go


register: build
	./$(BINARY_NAME) -config ./config.json -register


clean:
	rm -f $(BINARY_NAME) *~

