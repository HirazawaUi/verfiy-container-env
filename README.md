# verfiy-container-env

#### Used to test whether the CRI allows special characters as environment variable names

* Use the [CRI API](https://github.com/kubernetes/cri-api/blob/master/pkg/apis/runtime/v1/api.proto) to directly call the container runtime to create a Container, and set all printable ASCII characters (serial number 33-127) for the Container as the [environment variable name](https://github.com/HirazawaUi/verfiy-container-env/blob/558b7b0278668c4b8dded15e69c812d1d7b12e9f/main.go#L82-L89), and test whether the container runtime will do some special behavior for variable names with special characters

##### Runs with CRI-O

* The desired number of environment variables is 95

```
➜  ~ sudo ./verfiy-container-env -endpoint="unix:///run/crio/crio.sock"
I0128 19:01:34.489638  277439 main.go:78] The number of environment variables that have been set is 95
```

##### Runs with containerd

* The desired number of environment variables is 95

```
➜  ~ sudo ./verfiy-container-env
I0128 19:01:34.489638  277439 main.go:78] The number of environment variables that have been set is 95
```

##### Runs with cri-dockerd

...
