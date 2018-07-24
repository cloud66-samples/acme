# ACME Marketplace

```
version: 2
services:
  seller:
    git_url: https://github.com/cloud66-samples/acme.git
    git_branch: master
    dockerfile_path: "./Dockerfile"
    build_root: seller
    command: seller --redis redis:6379
  buyer:
    git_url: https://github.com/cloud66-samples/acme.git
    git_branch: master
    dockerfile_path: "./Dockerfile"
    build_root: buyer
    command: buyer --redis redis:6379
  dashboard:
    git_url: https://github.com/cloud66-samples/acme.git
    git_branch: master
    ports:
    - 5000:80
    dockerfile_path: "./Dockerfile"
    build_root: dashboard
    command: "/go/src/dashboard/dashboard --redis redis:6379"
  redis:
    image: redis
    ports:
    - '6379'
```
