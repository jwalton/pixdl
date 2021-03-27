# Providers

Providers are the "back ends" that pixdl can fetch images from.  There are lots of different types of image hosting services, so Provider has to be a little flexible in terms of how it works.

Broadly speaking, there are two kinds of providers.  The first are `URLProvider`s - providers that can download an album just given the URL.  For example, if the user gives us a URL like `https://imgur.com/gallery/88wOh`, then we can fetch `https://imgur.com/gallery/88wOh.json` and grab all the images for this album without ever having to parse any HTML.

The second kind of provider - `HTMLProvider` - is one that needs to scrape HTML.  Take a message board powered by XenForo as an example - the HTML will have a certain characteristic structure that's the same on all XenForo boards, so we can have a single XenForo provider that downloads images from all of them.  We could list all the URLs for known XenForo boards in that Provider, but if a URL isn't on our list, a Provider could still peek at the HTML and try to figure out if the structure is something it recognizes.

These two cases are different because in the first case, we just need to ask the Provider "Can you fetch albums from this URL?"  In the second case, the provider needs to look at parsed HTML, and obviously we don't want each provider to re-parse the HTML over and over again.  We therefore have two different Provider interfaces - URLProvider and HTMLProvider (although a given provider can implement both), and the overall algorithm for finding a provider is broadly:

* For each URLProvider, call `CanDownloadFromURL(url)`.  If the provider returns true, call `FetchAlbum()` to download the album.
* If no provider found, HEAD the URL
* If the URL is for an image, download the image directly.
* If the URL is for an HTML file, parse the HTML, and then for each HTMLProvider call `FetchAlbumFromHTML()` until one returns true, indicating that it found some images to download.

Note the last HTMLProvider is the "web" provider, which should be able to download just about anything.
