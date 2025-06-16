# Energy-aware kubernetes scheduler
 
This repository is for the Thesis of Athanasios Pyliotis for the Integrated Master's in Electrical and Computer Engineering in the National Technical University of Athens.

### Requirements
* Go: 1.24.4+
Required for building Kubernetes and the custom plugins.

* Docker: 27.5.1+
Used to build and push the custom scheduler image.

* Docker Hub account:
Needed for pushing and pulling the custom scheduler image. Remember to run docker login.

* MicroK8s: 1.32.4 revision 8148 +
Lightweight Kubernetes distribution used for running the local multi-node cluster.
Make sure the following MicroK8s addons are enabled:
dns, storage, metrics-server, helm3, rbac, hostpath-storage


## Contents
### Plugins folder
In the folder plugins the custom scoring plugins created for the thesis are included along with the updated registry.go . In order for them to properly work the installation of the kubernetes repository is required.


### Scheduler folder
In this folder the necessary yaml files for deploying a cystom kubernetes are included. Along with the Dockerfile for building the Docker image, a bash file for port forwarding in order to expose Prometheus metrics as well as a bash file to deploy everything in the proper order. Note that in order for these to run you need to change the dockerhub_user to your username in Docker Hub as well as have successfully ran docker login.

### Simulations
Contains the YAML files for the simulations that we performed in the scope of the Thesis.

### Rest
It also contains a bash file for annotating the necessary values to the nodes of the system. 
Finally, the Keppler-deployment is included because we needed to perform minor changes to it in order for Kepler to be exposed to prometheus. For it to work it is suggested to clone the Kepler repository and follow the official download process.
