# Stream directory content using TAR archive and HTTP

**WARNING**: This code is just for fun with tar and HTTP in go. Do not open this service for production or make it public.

## Sample usage

```
$ http-tar-dir /path/to/directory/to/stream USERNAME PASSWORD
```

When pointing your web browser to `http://localhost:8080`, you should be prompted for `USERNAME` and `PASSWORD` (specified in the arguments of the command).

After successfull authentication, you'll get access to a _TAR_ archive that is created on the fly with the content of the server's `/path/to/directory/to/stream` directory.
