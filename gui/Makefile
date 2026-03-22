# Build the GPX LiDAR GUI
# ele.py is embedded into the binary at build time via go:embed.
# It must be copied from the parent directory before building.

.PHONY: build clean

build:
	cp ../ele.py ele.py
	go build -o gpx-lidar-gui .

# Windows cross-compile (requires mingw-w64: brew install mingw-w64)
windows:
	cp ../ele.py ele.py
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
		go build -ldflags="-H windowsgui" -o gpx-lidar-gui.exe .

clean:
	rm -f gpx-lidar-gui gpx-lidar-gui.exe ele.py
