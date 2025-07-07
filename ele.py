#! /usr/bin/env python3
import os
import gpxpy
import rasterio
from pyproj import Transformer
import argparse
from geopy.distance import geodesic
from gpxpy.gpx import GPXTrackPoint

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

transformer = Transformer.from_crs("EPSG:4326", "EPSG:27700", always_xy=True)

def get_elevation(rasters, lon, lat):
    x, y = transformer.transform(lon, lat)
    for bounds, raster in rasters:
        if bounds.left <= x <= bounds.right and bounds.bottom <= y <= bounds.top:
            try:
                val = list(raster.sample([(x, y)]))[0][0]
                return float(val)
            except Exception as e:
                print(f"⚠️ Sampling error at ({x}, {y}): {e}")
                return None
    return None

def interpolate_points(p1, p2, max_spacing=1.0):
    start = (p1.latitude, p1.longitude)
    end = (p2.latitude, p2.longitude)
    distance = geodesic(start, end).meters

    if distance <= max_spacing:
        return []

    num_points = int(distance // max_spacing)
    points = []

    for i in range(1, num_points + 1):
        f = i / (num_points + 1)
        lat = p1.latitude + (p2.latitude - p1.latitude) * f
        lon = p1.longitude + (p2.longitude - p1.longitude) * f
        elev = None
        if p1.elevation is not None and p2.elevation is not None:
            elev = p1.elevation + (p2.elevation - p1.elevation) * f
        points.append(GPXTrackPoint(lat, lon, elevation=elev))
    return points

def densify_segment(segment, max_spacing=1.0):
    new_points = []
    for i in range(len(segment.points) - 1):
        p1 = segment.points[i]
        p2 = segment.points[i + 1]
        new_points.append(p1)
        new_points.extend(interpolate_points(p1, p2, max_spacing))
    new_points.append(segment.points[-1])
    segment.points = new_points

def tag_gpx_with_elevation(gpx_path, rasters, output_path, densify=False, max_spacing=1.0):
    with open(gpx_path) as f:
        gpx = gpxpy.parse(f)

    for track in gpx.tracks:
        for segment in track.segments:
            if densify:
                densify_segment(segment, max_spacing)
            for point in segment.points:
                elev = get_elevation(rasters, point.longitude, point.latitude)
                if elev is not None:
                    point.elevation = elev

    with open(output_path, 'w') as f:
        f.write(gpx.to_xml())
    print(f"✅ Saved tagged GPX to: {output_path}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Tag GPX file with elevation data from raster files.")
    parser.add_argument('-d', "--folder", type=str, required=True, help="Folder containing .tif raster files")
    parser.add_argument('-g', "--gpx_file", type=str, required=True, help="Input GPX file")
    parser.add_argument('-o', "--output_file", type=str, required=True, help="Output GPX file")
    parser.add_argument('--densify', action="store_true", help="Densify the GPX with interpolated points ≤ max spacing")
    parser.add_argument('--max_spacing', type=float, default=1.0, help="Max spacing (in meters) between GPX points when densifying")

    args = parser.parse_args()

    rasters = load_rasters(args.folder)
    for bounds, raster in rasters:
        print(f"{raster.name} CRS: {raster.crs}, Bounds: {bounds}")

    tag_gpx_with_elevation(
        args.gpx_file,
        rasters,
        args.output_file,
        densify=args.densify,
        max_spacing=args.max_spacing
    )
