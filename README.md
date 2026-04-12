# Kigo

A smart mirror implemented in Go, where modules communicate using publish-subscribe architecture. 
There are 2 fix modules, UI and Main. UI is responsible for drawing. Main is responsible to have the main logic, communicate between UI and modules (maybe modules communicate with the UI service directly).

I will use [GoGPU](https://github.com/gogpu/ui) as soon at is available. Nats for pubsub and my on lib sca-instruments for util and intrumentation, future REST API for external actors.

An additional package manager "kigoma" should be implemented which is able to add modules from public repository.

A list of modules implemented from [this list](https://github.com/MagicMirrorOrg/MagicMirror/wiki/3rd-Party-Modules):

| Module | Implemented |
| --- | --- |
| Text |  |
| Time |  |
| Calendar |  |
| Mail |  |
| Weather |  |
| Notifications |  |


