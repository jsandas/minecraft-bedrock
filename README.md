To execute the server using docker:

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
helm upgrade --install minecraft-bedrock oci://ghcr.io/jsandas/minecraft-bedrock -f custom-values.yaml -n minecraft --create-namespace
```
