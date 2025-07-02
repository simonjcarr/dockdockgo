# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A docker swarm like application that has the following features

- Written in GoLang
- A CLI with mirror API
- CLI can generate API tokens for secure API access
- The API will be used for external applications and GUI development

## User experience
The key to this application will be simplicity. It should be easy to simply install, specify an image, or docker-compose file, a list of remote servers and immediatley have a cluster of containers running in a highly available fashion. As much complexity and settings should be hidden from the user by using sensible defaults, but the user should be able to override important settings when required.

The API will set it appart from docker and docker swarm in order to allow easy remote management from other applications and web interfaces to be developed.

## Key Project Features
- Search remote registries for public and private images, if remote is not specified it should default to DockerHub
- Search for images locally
- User should be able to start a container from a local or remote image and specify all the settings they would by using the docker command
- The application should be able to read standard docker-compose files so it is backward compatible with docker compose
- When running a single container, the user should be able to provide a single remote server or list of remote servers to run the container on.
- The application should support an extended docker-compose file structure, where a list of remote servers can be provided for all or some of the containers to run on.
- The application should support the ability to provide an ssh key or username and password to connect to remote servers via SSH
- When connecting to a remote server, the app should check if docker is installed, if not it should prompt the user to confirm installing docker, but should also accept a switch to the dockdockgo command to install docker if not installed without asking.
- There will be option to specify how many replicas of the container should be run
- Where a container has port mappings specified, the app should ensure that their are no conflicts if the same container is run multiple times on the same server
- The app will install it's self on the remote machines along with zoo keeper. One instance of the app will be the master, and if not available another instance will take over as master.
- The each instance of the app on each server, will act as a tcp and http router supporting both http and https and will route traffic based on different policies (configurable) in order to share the traffic load between container instances. It should run an instance of redis in docker on each server in order to take care of managing instance availability, port mappings, and routing settings etc.
- The app should support using lets encrypt and self signed certificates for https

## progress tracking
This is your todo list of tasks to complete.
- if you find you new features you need to add, record them in PLAN.md. Ensure they go in the correct place in the file, so they implimented in the correct order.
- When you complete a feature ensure you check the item as complete in the PLAN.md file

## Documentation
You create a README.md file that contains instructions on how to use dockdockgo
The documentation should be simple and concise.
You should also create a folder with html pages of instructions and examples in the use of dockdockgo. There should be an index.html file that provides a getting started section and a links in logical order to the other html pages. This will eventually be used to create a support web site for the dockdockgo application.

