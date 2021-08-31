# Readme


## How to impelement cleanup?

We can list all our object inside of bucket , if files has a `expireAfter` tag and it's been expired we can send deletion request to s3.

Also we can configure our s3 cleanup sequence interval.