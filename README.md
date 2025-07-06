This script can be used to add more precise elevation data to GPX route files. 
The UK government published LIDAR Composite DTM at 1M detail which is freely available for download at https://environment.data.gov.uk/survey.

To use the script, download the tiles which cover your route and extract the zips into a directory, your tif folder should look like this once you extract the tile folders from the zips:
```fraser@MacBookAir elevationAdd % ls -alth tiffs
total 32
drwxr-xr-x@ 13 fraser  staff   416B  6 Jul 13:12 ..
drwxr-xr-x@ 42 fraser  staff   1.3K  6 Jul 13:03 .
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD86sw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD86nw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD76nw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD76ne
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD66nw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD66ne
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD57se
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD56sw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD56se
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD56nw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD56ne
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD46sw
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD46se
drwx------@  7 fraser  staff   224B  6 Jul 13:00 lidar_composite_dtm-2022-1-SD46ne
-rw-r--r--@  1 fraser  staff    10K  6 Jul 12:02 .DS_Store
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ54nw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ33se
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ52ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ64ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ54sw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ63nw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ64sw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ32ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ54ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ63ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ54se
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ53nw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ62nw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ53sw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ64se
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ63sw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ34ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ53ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ62ne
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ42nw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ53se
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ63se
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ64nw
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 lidar_composite_dtm-2022-1-TQ33ne
-rw-r--r--@  1 fraser  staff   1.2K  6 Jul 01:31 check_list.txt
fraser@MacBookAir elevationAdd % ls -alth tiffs/lidar_composite_dtm-2022-1-TQ54nw
total 90336
drwxr-xr-x@ 42 fraser  staff   1.3K  6 Jul 13:03 ..
drwxr-xr-x@  7 fraser  staff   224B  6 Jul 11:46 .
-rw-r--r--@  1 fraser  staff    44M  6 Jul 00:26 TQ54nw_DTM_1m.tif
-rw-r--r--@  1 fraser  staff   156K  6 Jul 00:26 TQ54nw_DTM_1m_Metadata.gpkg
-rw-r--r--@  1 fraser  staff    89B  6 Jul 00:26 TQ54nw_DTM_1m.tfw
-rw-r--r--@  1 fraser  staff   2.6K  6 Jul 00:26 TQ54nw_DTM_1m.tif.aux.xml
-rw-r--r--@  1 fraser  staff    20K  6 Jul 00:26 TQ54nw_DTM_1m.tif.xml
fraser@MacBookAir elevationAdd % ```


Then to apply the LIDAR elevation (or whichever other elevation map) used you can run the script:
./ele.py -d tiffs -g WOR-Day1\(CravenArms\).gpx -o WOR-Day1+LIDAR.gpx
...
Raster bounds: BoundingBox(left=380000.0, bottom=460000.0, right=385000.0, top=465000.0)
✅ Elevation found in tiffs/lidar_composite_dtm-2022-1-SD86sw/SD86sw_DTM_1m.tif: 138.77200317382812m at (-2.30096, 54.06198)
✅ Saved tagged GPX to: WOR-Day1+LIDAR.gpx


Example options:
(.venv) fraser@MacBookAir elevationAdd % ./ele.py -h                                                          
usage: ele.py [-h] -d FOLDER -g GPX_FILE -o OUTPUT_FILE

Tag GPX file with elevation data from raster files.

options:
  -h, --help            show this help message and exit
  -d, --folder FOLDER   Folder containing .tif raster files. .tif can be in subfolders of the specified folder.
  -g, --gpx_file GPX_FILE
                        Input GPX file to tag with elevation
  -o, --output_file OUTPUT_FILE
                        Output GPX file with elevation data
(.venv) fraser@MacBookAir elevationAdd % 

After running you can check the output gpx (its similar to an xml file) It will use the closest meter elevation from the map (if you use the 1m LIDAR DTM as in the example) which is much more accurate in most cases than the data which nasa, google, OSM provides!

