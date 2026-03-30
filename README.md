# Kigo

A smart mirror implemented in Go, where modules communicate using publish-subscribe architecture. 
There are 2 fix modules, UI and Main. UI is responsible for drawing. Main is responsible to have the main logic, communicate between UI and modules (maybe modules communicate with the UI service directly).

Core modules include calendar, time, weather, text and email.

I will use [GoGPU](https://github.com/gogpu/ui) as soon at is available. Nats for pubsub and my on lib sca-instruments for util and intrumentation, future REST API for external actors.
