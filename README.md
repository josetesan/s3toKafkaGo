S3Uploader
===

1. Needs to have AWS credentials in enviroment
```bash
export AWS_ACCESS_KEY_ID=<your key>
export AWS_SECRET_ACCESS_KEY=<your secret>
```

2. Sends a file with curl
```bash
curl -X PUT http://localhost:3000/ -F "file=@<file>"
```