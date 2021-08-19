# pixdl

pixdl is a tool for downloading images from online galleries.  Currently, it supports:

* imgur.com
* gofile.io
* XenForo powered forums
* Any web page with lots of images on it

## Features

* Downloads multiple files in parallel.
* Resumes downloads if interrupted.
* Automatic retries on failure.
* Skips files that have already been downloaded.
* Shows progress while downloading.

## Usage

```sh
# Download files into the current directory
pixdl get https://imgur.com/gallery/88wOh

# Download files into another directory
pixdl get -o ./album https://imgur.com/gallery/88wOh

# Download files and sort them into subfolders based on which post they were in
pixdl get -o ./bikes --template "{{.Image.SubAlbum}}/{{.Filename}}" https://www.cyclechat.net/threads/four-of-my-carlton-bikes.273364/

# Download files from the first page of a XenForo forum
pixdl get -o ./bikes --max-pages 1 https://www.cyclechat.net/threads/four-of-my-carlton-bikes.273364/

# Download only images from post #22
pixdl get -o ./bikes --subalbum 22 https://www.cyclechat.net/threads/four-of-my-carlton-bikes.273364/
```
