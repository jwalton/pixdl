package imgur

type imgurGalleryResponse struct {
	Data struct {
		Image struct {
			ID          int64  `json:"id"`
			Hash        string `json:"hash"`
			AccountURL  string `json:"account_url"`
			Title       string `json:"title"`
			AlbumImages struct {
				Count  int64        `json:"count"`
				Images []imgurImage `json:"images"`
			} `json:"album_images"`
		} `jason:"image"`
	} `json:"data"`
}

type imgurImage struct {
	// Hash is a unique ID.  Can download the actual image from `https://i.imgur.com/${Hash}.${Ext}`.
	Hash string `json:"hash"`
	// Title is the title of the file, but is often empty.
	Title string `json:"title"`
	// Size is the size of the image, in bytes.
	Size int64 `json:"size"`
	// Description is a description of the file.
	Description string `json:"description"`
	// Name is the name of the file, without the extension.
	Name string `json:"name"`
	// Ext is the extension of the file.
	Ext string `json:"ext"`
	// Datetime is in format "2017-07-31 12:25:20".
	Datetime string `json:"datetime"`
}
