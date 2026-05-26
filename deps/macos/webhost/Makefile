BINARY_NAME = WebHost
BUNDLE_ID   = com.dolphin.webhost
BUILD_DIR   = .build
APP_DIR     = $(BINARY_NAME).app

.PHONY: all build app run clean

all: app

build:
	swift build -c release

app: build
	rm -rf "$(APP_DIR)"
	mkdir -p "$(APP_DIR)/Contents/MacOS"
	mkdir -p "$(APP_DIR)/Contents/Resources"
	cp "$(BUILD_DIR)/release/$(BINARY_NAME)" "$(APP_DIR)/Contents/MacOS/$(BINARY_NAME)"
	sed -e "s/BUNDLE_ID/$(BUNDLE_ID)/g" -e "s/BINARY_NAME/$(BINARY_NAME)/g" Info.plist.tmpl > "$(APP_DIR)/Contents/Info.plist"
	touch "$(APP_DIR)"
	@echo "Built $(APP_DIR)"

run: app
	open "$(APP_DIR)"

clean:
	rm -rf "$(APP_DIR)"
	swift build --clean 2>/dev/null || true
