# github-as-s3

> This post is an educational exploration into the S3 protocol and a practical guide on how one might implement it using Git as the underlying storage mechanism.
> Please be mindful of the terms of service and code of conduct for any Git hosting provider you choose to use with these concepts; I am not responsible for any actions taken against your accounts.

![a picture of git pretending to be s3](https://kristun.dev/_astro/s3-git-scooby-doo.D9vfHZvy_ZyPgXk.webp)

This project contains the source code for the Github+S3 proxy.

The intended use is to proxy object storage onto Git itself. You can read more about this [here](https://kristun.dev/posts/git-as-s3/)

The following routes are implemented

- [PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html)
- [GetObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html)
- [HeadObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html)
- [DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html)
- [ListObjectsV2](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html)
- [CreateBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html)
- [DeleteBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucket.html)
- [ListBuckets](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBuckets.html)

# Getting started

## Run it with Go

```bash
cp .env.example .env
go run ./cmd/cli/
```

The environment values require the following:

```bash
# required - replace with your own username and token
GITHUB_TOKEN=
GITHUB_OWNER=ktunprasert

# optional
# hosting related
GHS3_PORT=
GHS3_ADDRESS=
# committer's information
# these are the defaults defined in application.go
GIT_USERNAME=GHS3
GIT_EMAIL=bot@ghs3.com
```

The token requires 2 permissions: `repo`, `delete_repo`

## Run it with Docker

```bash
docker build . -t ghs3
docker run --rm -t ghs3 --env "GITHUB_OWNER=ktunprasert" --env "GITHUB_TOKEN=$TOKEN"
```

## Run it with docker-compose

```bash
# configure your compose file with proper environment values
docker-compose up -d
```

## Development

```bash
air
```

Hot reloading with [`air`](https://github.com/air-verse/air) is configured

# Compatible & Tested applications

| Tool                                  | Tested                                                            |     |
| ------------------------------------- | ----------------------------------------------------------------- | --- |
| [rclone](https://rclone.org/)         | cp, sync, delete, mkdir, purge, ls, deletefile, touch, lsd, rmdir | ✅  |
| [aws s3](https://aws.amazon.com/cli/) | mb, cp, ls, rm                                                    | ✅  |
| [pocketbase](https://pocketbase.io/)  | creating and restoring back up + deleting files                   | ✅  |
| [pocketbase](https://pocketbase.io/)  | using as file storage                                             | ❓  |
