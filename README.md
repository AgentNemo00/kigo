# Kigo

A smart mirror implemented in Go, where modules communicate using pubsub architecture. 
There are 2 fix processes, `KiGoUI` and `KiGo`. The `KiGoUI` is responsible for handling the screen(s) and drawing. KiGo is responsible to have the logic responsible to handle/manage modules and failures. Modules are third party processes with logic about what to be shown on the screens. A manager `KiGoMa` is responsible for downloading third party modules from any repository (Schema needs to be implemented) handling integration via REST api.
Modules can be run everywhere you want, `KiGoUI` and `KiGo` (for now) are running on the device connected to the screen. The modules can be run on the same device or at a remote devices. This gives us the benefit of expanding heavy loads, i.e. ML/AI/SMLs (your choice). For a distributed architecture some changes would need to me made. `KiGo` will have a REST API which is currently (only ping and metrics) unfineshed and should integrate `KiGoMa`.
The scope of `KigoMa` would be to download and manage modules.



I will use [GoGPU](https://github.com/gogpu/ui) as soon at is available for the `KiGoUI`. [Nats](https://nats.io/) for pubsub and my on lib sca-instruments for util and intrumentation, future REST API for external actors. Shared memory for drawing/sharing 

[KiGoCore](https://github.com/AgentNemo00/kigo-core) has the current definition of the inter process messages.


## Architecture

![Architecture](assets/architecture.png)

`KiGo` is responsible for handling the module lifecycle and `KiGoUI` for rendering what the modules want to draw. For more informations about the module lifecycle checkout
[KiGoCore](https://github.com/AgentNemo00/kigo-core).

## Roadmap

### Phase 0

- [ ] Define architecture
- [ ] Define communication shema via pubsub
- [ ] Define communication via IPC

### Phase 1

- [ ] Implement communication between KiGoUI and KiGo
- [ ] Implement communication between KiGo and modules via pubsub

### Phase 2

- [ ] Implement modules render roundtrip
- [ ] Implement basic modules

### Phase 3

- [ ] Implement deployment architecture, including KiGo, KiGoUI and the pubsub server
- [ ] Implement basic modules

### Phase 4

- [ ] Implement KiGoMa on device

### Phase 5

- [ ] Implement KigoMa on remote
- [ ] Implement remote modules

A list of modules implemented from [this list](https://github.com/MagicMirrorOrg/MagicMirror/wiki/3rd-Party-Modules):

| Module | Implemented |
| --- | --- |
| Text |  |
| Time |  |
| Calendar |  |
| Mail |  |
| Weather |  |
| Notifications |  |

