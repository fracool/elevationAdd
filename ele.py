#! /usr/bin/env python3
import os
import gpxpy
import rasterio
from pyproj import Transformer
import argparse

"""This script loads elevation data from .tif raster files, transforms coordinates from WGS84 to British National Grid (EPSG:27700),"""
"""and tags a GPX file with elevation data based on the closest raster."""
"""Get rasters from https://environment.data.gov.uk/survey"""

# Load all .tif rasters and their bounds
def load_rasters(folder):
    rasters = []
    for dirpath, _, filenames in os.walk(folder):
        for filename in filenames:
            if filename.endswith(".tif") and not filename.startswith("."):
                path = os.path.join(dirpath, filename)
                try:
                    raster = rasterio.open(path)
                    rasters.append((raster.bounds, raster))
                    print(f"Loaded: {path}")
                except Exception as e:
                    print(f"❌ Failed to load {path}: {e}")
    print(f"✅ Total rasters loaded: {len(rasters)}")
    return rasters



# Transform coordinates from WGS84 to EPSG:27700 (British National Grid)
transformer = Transformer.from_crs("EPSG:4326", "EPSG:27700", always_xy=True)

# Find elevation from correct raster
def get_elevation(rasters, lon, lat):
    x, y = transformer.transform(lon, lat)
    print(f"Transformed ({lon}, {lat}) to BNG (x={x:.2f}, y={y:.2f})")

    found_raster = False
    for bounds, raster in rasters:
        print(f"Checking raster: {raster.name}")
        print(f"Raster bounds: {bounds}")
        if bounds.left <= x <= bounds.right and bounds.bottom <= y <= bounds.top:
            found_raster = True
            try:
                val = list(raster.sample([(x, y)]))[0][0]
                print(f"✅ Elevation found in {raster.name}: {val}m at ({lon}, {lat})")
                return float(val)
            except Exception as e:
                print(f"⚠️ Sampling error at ({x}, {y}) in {raster.name}: {e}")
                return None

    if not found_raster:
        print(f"❌ No raster bounds match for ({x:.2f}, {y:.2f}) from ({lon}, {lat})")
    return None


# Update GPX file with elevation
def tag_gpx_with_elevation(gpx_path, rasters, output_path):
    with open(gpx_path) as f:
        gpx = gpxpy.parse(f)

    for track in gpx.tracks:
        for segment in track.segments:
            for point in segment.points:
                elev = get_elevation(rasters, point.longitude, point.latitude)
                if elev is not None:
                    point.elevation = elev

    with open(output_path, 'w') as f:
        f.write(gpx.to_xml())
    print(f"✅ Saved tagged GPX to: {output_path}")



# Run
if __name__ == "__main__":

    parser = argparse.ArgumentParser(description="Tag GPX file with elevation data from raster files.")
    parser.add_argument('-d', "--folder", type=str, help="Folder containing .tif raster files. .tif can be in subfolders of the specified folder.", required=True)
    parser.add_argument('-g', "--gpx_file", type=str, help="Input GPX file to tag with elevation", required=True)
    parser.add_argument('-o', "--output_file", type=str, help="Output GPX file with elevation data", required=True)

    args = parser.parse_args()
    if args.folder:
        folder = args.folder
    else:
        folder = "./tiffs"  # default folder


    rasters = load_rasters(folder)
    for bounds, raster in rasters:
        print(f"{raster.name} CRS: {raster.crs}, Bounds: {bounds}")

    tag_gpx_with_elevation(args.gpx_file, rasters, args.output_file)
