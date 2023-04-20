package constant

import "errors"

var ErrFileExists = "file exists"
var LinkNotFound = "Link not found"

var NDPFoundReply error = errors.New("found ndp reply")
var NDPFoundError error = errors.New("found err")
var NDPRetryError error = errors.New("ip conflicting check fails with more than maximum number of retries")
