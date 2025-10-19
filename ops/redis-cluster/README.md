- ネットワーク作成
```bash
docker network create redis-cluster-network
```
- ビルド
```bash
docker compose build
```
- コンテナ起動
```bash
docker compose up -d --scale redis=6 
```
- 各RedisのIPアドレスを確認
```bash
docker network inspect redis-cluster_redis-cluster-network | jq '.[0].Containers | .[].IPv4Address'
```
- クラスター作成
```bash
docker compose exec redis redis-cli --cluster create 172.25.0.7:6379 172.25.0.4:6379 172.25.0.2:6379 172.25.0.5:6379 172.25.0.3:6379 172.25.0.6:6379 --cluster-replicas 1
```
- 動作確認
```bash
docker compose exec redis redis-cli -c -h 172.25.0.7

ex）
172.25.0.7:6379> set foo bar
-> Redirected to slot [12182] located at 172.25.0.2:6379
OK
172.25.0.2:6379> get foo
"bar"
172.25.0.2:6379> set hello world
-> Redirected to slot [866] located at 172.25.0.7:6379
OK
172.25.0.7:6379> get hello
"world"
172.25.0.7:6379> get foo
-> Redirected to slot [12182] located at 172.25.0.2:6379
"bar"
```