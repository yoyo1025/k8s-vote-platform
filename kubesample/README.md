# Kubernetesサンプル
## MiniKubeで起動する場合
1. Docker の宛先を Minikube に切り替える（毎シェルで最初に1回実行）
```bash
eval $(minikube docker-env)
```

2. ビルド
```bash
docker build -t kube-sample-app:latest . --no-cache
```
3. アプライ
```bash
$ kubectl apply -f .
```

4. ロールアウト（更新時）
```bash
kubectl -n kubesample rollout restart deploy/kubesample-deployment
```

## Docker DesktopでKubernetesを起動する場合
### Dockerイメージビルド
`k8s-vote-platform/kubesample` にて
1. ビルド
```bash
$ docker build . -t kube-sample-app
```
2. 確認
```bash
$ docker images

REPOSITORY                                TAG                                                                           IMAGE ID       CREATED          SIZE
kube-sample-app                           latest                                                                        99417e5c6912   46 minutes ago   1.51GB
```

### マニフェストをKubernetesにデプロイ
`k8s-vote-platform/kubesample/manifest` にて
1. マニフェストをKubernetesにデプロイ
```bash
$ kubectl apply -f .
```
2. 確認
```bash
$ kubectl get ns

NAME              STATUS   AGE
default           Active   16m
kube-node-lease   Active   16m
kube-public       Active   16m
kube-system       Active   16m
kubesample        Active   14m
```
### 動作確認
- service確認
```bash
$ kubectl get service -n kubesample

NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
kubesample-service   ClusterIP   10.103.21.224   <none>        3000/TCP   16m
```
- deployment確認
```bash
$ kubectl get deployment -n kubesample

NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
kubesample-deployment   3/3     3            3           16m
```
- ポートフォワード&テスト実行
```bash
$ kubectl port-forward service/kubesample-service -n kubesample 3000:3000

Forwarding from 127.0.0.1:3000 -> 4883
Forwarding from [::1]:3000 -> 4883
Handling connection for 3000
```
```bash
$ curl http://localhost:3000/ping

{"message":"pong"}
```