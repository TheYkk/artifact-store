# Open Questions
## How to maximize the security of the service in general and client => service => third-party interactions?
First we can run the service inside the private network.
## How to make sure we don't run out of space / cleanup the artifacts?
In this implementation we use s3, we didn't use local storage . So we can safe about local storage usage. But if we are talking about s3 storage usage ; s3 has own cleanup interval.
So it's delete files after that interval.
## How to make sure we're always getting the newest version of the file?
I set an hour caching allowance to artifacts, when the object tried to pull after 1 hour we are requesting the source artifact and check the Etag of the object.
I used Etag comparison between source file and in s3 file. 