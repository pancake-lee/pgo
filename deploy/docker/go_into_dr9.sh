#!/bin/sh
docker exec -it -w /backend/ pgo-dr9 /bin/bash

# window上用git-bash似乎路径转换有问题，进去再切换路径
# docker exec -it pgo-dr9 bash
