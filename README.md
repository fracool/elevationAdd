# elevationAdd

Add precise elevation data to GPX route files using UK government LIDAR DTM (Digital Terrain Model) raster tiles.

The UK government publishes LIDAR Composite DTM data at 1-metre resolution, freely available at
https://environment.data.gov.uk/survey. This script reads those GeoTIFF tiles and stamps each point in a GPX file
with the corresponding ground elevation — much more accurate than GPS-recorded or cloud-based sources (Google, NASA, OSM).

> **Note:** The raster data must use the British National Grid coordinate system (EPSG:27700), so this tool is designed for UK coverage.

## Requirements

- Python 3
- pip packages:

```
pip3 install gpxpy rasterio pyproj geopy
```

Or install from the included `requirements.txt`:

```
pip3 install -r requirements.txt
```

## Usage

```
./ele.py -d <raster_folder> -g <input.gpx> -o <output.gpx> [--densify] [--max_spacing <meters>]
```

| Flag | Description |
|------|-------------|
| `-d`, `--folder` | Folder containing `.tif` raster files (searched recursively) |
| `-g`, `--gpx_file` | Input GPX file to tag with elevation |
| `-o`, `--output_file` | Output GPX file with elevation data |
| `--densify` | Insert extra points between existing track points so the elevation profile has finer detail |
| `--max_spacing` | Maximum distance in metres between points when densifying (default: `1.0`) |

### Preparing raster data

Download the LIDAR tiles that cover your route from the link above and extract the zip files into a single directory. The script walks sub-folders automatically, so the layout can look like this:

```
tiffs/
├── lidar_composite_dtm-2022-1-SD86sw/
│   └── SD86sw_DTM_1m.tif
├── lidar_composite_dtm-2022-1-SD86nw/
│   └── SD86nw_DTM_1m.tif
└── ...
```

### Example

```bash
./ele.py -d tiffs -g WOR-Day1\(CravenArms\).gpx -o WOR-Day1+LIDAR.gpx
```

```
Loaded: tiffs/lidar_composite_dtm-2022-1-SD86sw/SD86sw_DTM_1m.tif
...
✅ Total rasters loaded: 14
✅ Saved tagged GPX to: WOR-Day1+LIDAR.gpx
```

### Densify example

If your GPX track has long gaps between points, `--densify` will interpolate additional points
at up to `--max_spacing` metres apart so the elevation profile captures more terrain detail:

```bash
./ele.py -d tiffs -g route.gpx -o route_dense.gpx --densify --max_spacing 2.0
```

## How it works

1. All `.tif` raster files in the specified folder (and sub-folders) are loaded.
2. GPS coordinates (WGS 84 / EPSG:4326) are transformed to British National Grid (EPSG:27700).
3. For each track point the script finds the raster tile whose bounding box contains the point and samples the elevation at the nearest pixel.
4. If `--densify` is enabled, extra points are linearly interpolated between existing track points before elevation tagging.
5. The result is written to a new GPX file.

## Limitations

- Only supports raster tiles in EPSG:27700 (UK LIDAR data).
- Elevation is sampled from the single nearest pixel; no sub-pixel interpolation is performed.
- Densification uses linear interpolation of latitude/longitude, which is acceptable for short distances but less precise over very long segments.
