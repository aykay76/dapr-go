To connect to Redis CLI:
docker exec -it dapr_redis redis-cli

To clear down the stream `core`:
xlen core
xtrim core maxlen 0
