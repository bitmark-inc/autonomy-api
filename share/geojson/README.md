# GeoJSON Data

This is a directory that contains all geojson data required by this project.

## Tools

We use tools inside [Geospatial Data Abstraction Library](https://www.gdal.org/) to extract geojson from boundary data in SHP format.

### Installation

Mac: `brew install gdal`

## Data Extraction

### Taiwan Boundary

1. Download [open data](https://data.gov.tw/dataset/7442)
2. Unzip it. There will be a SHP file **COUNTY_MOI_1081121.shp** (file name might be changed in the future)
3. Use `ogr2ogr` to convert it into geojson with WGS84 coordinates (regular latitude and longitude):
    ```
    ogr2ogr -f "GeoJSON" -s_srs EPSG:3824 -t_srs EPSG:4326 tw-boundary.json COUNTY_MOI_1081121.shp
    ```

### World Country Boundary

1.Download [open data](https://hub.arcgis.com/datasets/252471276c9941729543be8789e06e12_0)
2. Unzip it. There will be a SHP file **World_Countries__Generalized_.shp** (file name might be changed in the future)
3. Use `ogr2ogr` to convert it into geojson with WGS84 coordinates (regular latitude and longitude):
    ```
    ogr2ogr -f "GeoJSON" -t_srs EPSG:4326 world-boundary.json World_Countries__Generalized_.shp
    ```

### US Boundary

1. Download GeoJSON file [open data](https://public.opendatasoft.com/explore/dataset/us-county-boundaries/export/)

## Import boundary data to DB

```
# export AUTONOMY_MONGO_DATABASE='autonomy'
# export AUTONOMY_MONGO_CONN='mongodb://127.0.0.1:27017/?compressors=disabled'
# go run import-boundary/main.go
```

## References

- https://gis.stackexchange.com/questions/86153/in-ogr2ogr-what-is-srs
