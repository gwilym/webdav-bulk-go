webdav-bulk-go
==============

This is a proof-of-concept of a bulk webdav transfer tool in Go.

Written to test some internal systems. This isn't consumer-friendly and probably
shouldn't be used if you value your files.

See `LICENSE` for (lack of) warranty details.

Uploaded publicly for the interest of Go programmers since it's a working
example of bulk HTTP operations supporting Digest authentication. It's probably
terrible in terms of both implementation and style so copy-pasta with caution if
you intend to use this in something important.

It advertises download capabilities but only upload is implemented so far.

Usage
-----

`webdav-bulk-go -h`

Uploading...

`webdav-bulk-go /foo/bar https://user:pass@host/dav/foo/bar`
