package gofile

type gofileUpload struct {
	// Status is the status of the request - should be "ok".
	Status string `json:"status"`
	// Data is the data for the request.
	Data struct {
		// Code is the unique ID for this album.
		Code string `json:"code"`
		// CreateTime is the unix timestamp this file was created at (e.g. 1618941563)
		CreateTime int64 `json:"createTime"`
		// TotalDownload is the number of times this album has been downloaded.
		TotalDownload int64 `json:"totalDownloadCount"`
		// TotalSize is the total size of all items in this album, in bytes.
		TotalSize int64 `json:"totalSize"`
		// Files is a hash of all files in this album, indexed by md5 hash.
		Files map[string]gofileFile `json:"contents"`
	} `json:"data"`
}

type gofileFile struct {
	// Name is the filename for this file.
	Name string `json:"name"`
	// Size is the size of this file, in bytes.
	Size int64 `json:"size"`
	// Mimetype is the MIME type for this file.
	Mimetype string `json:"mimetype"`
	// Link is the URL to download this file from.
	Link string `json:"link"`
}
