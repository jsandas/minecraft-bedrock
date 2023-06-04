Run minecraft bedrock edition server in a container with docker/docker-compose or kubernetes/helm.  

The version of the server is static in the repo and is updated via the `Version Check` Github Action.

**Docker**

The first time: (it creates a docker container named minecraft-server, this command works only once)
`docker-compose run --name minecraft-server server`

To return to the terminal and keep the server running:
Keep holding CTRL, press P, release P, press Q, release Q, and release CTRL

Starting interactively (CTRL+C will stop it, use the escape sequence to detach without stopping it)
`docker start minecraft-server -i`

Starting seeing the log (CTRL+C will not stop it but you won't be able to run commands)
`docker start minecraft-server -a`

Starting in the background
`docker start minecraft-server`

Opening the server console which were running in background: (you won't see previous log messages, just tart typing commands)
`docker attach minecraft-server`

Stopping the server that is running in background:
`docker stop minecraft-server`

To remove the container that you have created (the data folder will not be deleted, it must be stopped)
`docker rm minecraft-server`

**Kubernetes**

Install:
```
helm upgrade --install <release_name> oci://ghcr.io/jsandas/minecraft-bedrock
```
Example install with specific namespace and custom values file:
```
helm upgrade --install minecraft-bedrock oci://ghcr.io/jsandas/minecraft-bedrock -f custom-values.yaml --namespace minecraft --create-namespace
```

To manage minecraft server (assuming a single deployment of minecraft per namespace):
```
export POD=$(kubectl get -n <namespace> pods | grep -v NAME | cut -d " " -f1)
kubectl exec -n <namespace> -it $POD -- bash -c "./mccli"
```
Example:
```
export POD=$(kubectl get -n minecraft pods | grep -v NAME | cut -d " " -f1)
kubectl exec -n minecraft -it $POD -- bash -c "./mccli"
```
The mccli behaves similar to the standard minecraft console.  Commands such as `help` or `gamerule` can run.  When done use `ctrl+c` to exit mccli.

Notes:
Clients will need to be configured to connect to the server's ip address or hostname.  The minecraft server application is unable to broadcast outside of the kubernetes network.