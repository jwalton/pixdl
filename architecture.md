# Architecture

Web albums come in many shapes and sizes. Some albums provide a nice API for fetching this, some we have to crawl a web page.  Sometimes we can get all the images in a single request, sometimes we need to make multiple requests.

We want to support the following features:

* Automatically download new images/resume an interrupted download from an album we have downloaded before.
* If downloading multiple albums, sort albums into folders based on album name (or other metadata).
* Download multiple images/albums concurrently.

When you run `pixdl [url]` the following happens:

* For each URL, we find a `AlbumDownloader` instance which can fetch data from the album.  These are found in pkg/downloaders.  Then, for each AlbumDownloader:
* We call into `AlbumDownloader.FetchAlbumData()` to fetch metadata (and any images we can find from the initial web request).
* We check to see if we have a subdirectory already for the album.
* In a loop, we call into `AlbumDownloader.FetchMoreImages()` to get URLs for images to download.  For each image, we check to see if we already have the image or not, and then either download the image or skip it.