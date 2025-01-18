# Changelog

## 0.1.0 (2025-01-18)


### Features

* add fake host client implementation ([#91](https://github.com/karelvanhecke/libvirt-operator/issues/91)) ([c68f1ab](https://github.com/karelvanhecke/libvirt-operator/commit/c68f1aba1f9fc60fa0e7b628ace7afa811733f91))
* **api:** add cloud-init API ([#139](https://github.com/karelvanhecke/libvirt-operator/issues/139)) ([34bc6fe](https://github.com/karelvanhecke/libvirt-operator/commit/34bc6fe0b38e8b9da97cbe81ffbc910c5a63ff9d))
* **api:** add domain API ([#179](https://github.com/karelvanhecke/libvirt-operator/issues/179)) ([a95964c](https://github.com/karelvanhecke/libvirt-operator/commit/a95964c7ac50f34a7017712b084b5f369b1b73f9))
* **api:** add reference only data objects ([#153](https://github.com/karelvanhecke/libvirt-operator/issues/153)) ([decb8d4](https://github.com/karelvanhecke/libvirt-operator/commit/decb8d4bf256cc9b7cf5c895a4b6149660e6e40b))
* **api:** return namespaced name for resource refs ([#216](https://github.com/karelvanhecke/libvirt-operator/issues/216)) ([61d69f3](https://github.com/karelvanhecke/libvirt-operator/commit/61d69f3e7a4cfb4f12adb1d22dd2e555c857d026))
* **domain:** allow setting the interface mac address ([#191](https://github.com/karelvanhecke/libvirt-operator/issues/191)) ([d34b5e2](https://github.com/karelvanhecke/libvirt-operator/commit/d34b5e26c4765feff31f46535458bf9f48ddc30d))
* generate unique names for libvirt resources ([#200](https://github.com/karelvanhecke/libvirt-operator/issues/200)) ([608db3b](https://github.com/karelvanhecke/libvirt-operator/commit/608db3b84ebe43dd04afc391f324ab578117daf8))
* **host:** update host interface and implement in fake host ([#150](https://github.com/karelvanhecke/libvirt-operator/issues/150)) ([e50476c](https://github.com/karelvanhecke/libvirt-operator/commit/e50476c5ccd7e0654408db9eccdc545e35d1acf9))
* initial libvirt operator implementation ([83fd20b](https://github.com/karelvanhecke/libvirt-operator/commit/83fd20b56c3af41baa66960b4fd028a39b14439a))
* make conditions, labels and finalizers part of the API ([#177](https://github.com/karelvanhecke/libvirt-operator/issues/177)) ([5473f65](https://github.com/karelvanhecke/libvirt-operator/commit/5473f654c4d1baa559e6e86588fa407e6126a61f))
* make external resource name part of API ([#212](https://github.com/karelvanhecke/libvirt-operator/issues/212)) ([0730918](https://github.com/karelvanhecke/libvirt-operator/commit/073091897aa7969aeb7dbb54b12e67d394552d95))
* **volume:** live resize ([#189](https://github.com/karelvanhecke/libvirt-operator/issues/189)) ([c304f7e](https://github.com/karelvanhecke/libvirt-operator/commit/c304f7ee03d3cb2dcad0b78878e651ec6a6a49ed))


### Bug Fixes

* **action/domain:** fix qemu guest-agent channel ([#183](https://github.com/karelvanhecke/libvirt-operator/issues/183)) ([f8a256a](https://github.com/karelvanhecke/libvirt-operator/commit/f8a256af5d6bef60d687a2c511c42a1abea4edb3))
* **action/volume:** fallback to regular resize method during live resize ([#214](https://github.com/karelvanhecke/libvirt-operator/issues/214)) ([06889f6](https://github.com/karelvanhecke/libvirt-operator/commit/06889f675df4c6af0cb6bba5ea25f462877ed9ed))
* **api:** correct typo in VolumeList metadata json struct tag ([#111](https://github.com/karelvanhecke/libvirt-operator/issues/111)) ([76e0f28](https://github.com/karelvanhecke/libvirt-operator/commit/76e0f286c5f5279ec751a44dd84ca3a07fb1944f))
* **deps:** update github.com/digitalocean/go-libvirt digest to 901e01e ([#92](https://github.com/karelvanhecke/libvirt-operator/issues/92)) ([bf5b513](https://github.com/karelvanhecke/libvirt-operator/commit/bf5b513fb4aabd2cf4c8ff4f7619690bfb7cae92))
* **deps:** update github.com/digitalocean/go-libvirt digest to 9fbdb61 ([#108](https://github.com/karelvanhecke/libvirt-operator/issues/108)) ([0ee7fdf](https://github.com/karelvanhecke/libvirt-operator/commit/0ee7fdfe722abaf75b81f887c67f45e75d9526a3))
* **deps:** update github.com/digitalocean/go-libvirt digest to c54891a ([#60](https://github.com/karelvanhecke/libvirt-operator/issues/60)) ([d5b0136](https://github.com/karelvanhecke/libvirt-operator/commit/d5b01369b763d693612bd015b3739c798b99cb36))
* **deps:** update kubernetes packages to v0.31.2 ([#7](https://github.com/karelvanhecke/libvirt-operator/issues/7)) ([68f1742](https://github.com/karelvanhecke/libvirt-operator/commit/68f17421dd512662354c329da54862950d2e275c))
* **deps:** update kubernetes packages to v0.31.3 ([#73](https://github.com/karelvanhecke/libvirt-operator/issues/73)) ([873335c](https://github.com/karelvanhecke/libvirt-operator/commit/873335c45f3f6d050d0c81a9059b5d0c9c849e6b))
* **deps:** update kubernetes packages to v0.31.4 (patch) ([#104](https://github.com/karelvanhecke/libvirt-operator/issues/104)) ([e4f9270](https://github.com/karelvanhecke/libvirt-operator/commit/e4f9270f8bb3aaa778bfccbd4ab2df63af843fcb))
* **deps:** update kubernetes packages to v0.31.5 (patch) ([#205](https://github.com/karelvanhecke/libvirt-operator/issues/205)) ([d96f119](https://github.com/karelvanhecke/libvirt-operator/commit/d96f119e2230923dadf94b785053350eb5f88326))
* **deps:** update module github.com/arm-software/golang-utils/utils to v1.77.1 ([#172](https://github.com/karelvanhecke/libvirt-operator/issues/172)) ([17576e5](https://github.com/karelvanhecke/libvirt-operator/commit/17576e55bbad0c491d4ba2fad01880bbd37259e7))
* **deps:** update module github.com/arm-software/golang-utils/utils to v1.79.0 ([#173](https://github.com/karelvanhecke/libvirt-operator/issues/173)) ([cc8b093](https://github.com/karelvanhecke/libvirt-operator/commit/cc8b093bb0653d7497a60353f3734052d1f6eec5))
* **deps:** update module github.com/arm-software/golang-utils/utils to v1.80.0 ([#196](https://github.com/karelvanhecke/libvirt-operator/issues/196)) ([e7fe565](https://github.com/karelvanhecke/libvirt-operator/commit/e7fe565ed95141f6b65b9f1c7edb87ebf8e0b592))
* **deps:** update module libvirt.org/go/libvirtxml to v1.10009.0 ([#11](https://github.com/karelvanhecke/libvirt-operator/issues/11)) ([ccd7a4b](https://github.com/karelvanhecke/libvirt-operator/commit/ccd7a4b5000b3a77e41ad11634ca4e5f93fc5a29))
* **deps:** update module libvirt.org/go/libvirtxml to v1.10010.0 ([#169](https://github.com/karelvanhecke/libvirt-operator/issues/169)) ([4c33e63](https://github.com/karelvanhecke/libvirt-operator/commit/4c33e6363c645d46808f6031d2c0330551200942))
* **deps:** update module libvirt.org/go/libvirtxml to v1.11000.0 ([#208](https://github.com/karelvanhecke/libvirt-operator/issues/208)) ([7757280](https://github.com/karelvanhecke/libvirt-operator/commit/775728085fb934b1b535a36f41291ee061cc7070))
* **deps:** update module libvirt.org/go/libvirtxml to v1.11000.1 ([#218](https://github.com/karelvanhecke/libvirt-operator/issues/218)) ([252f8f5](https://github.com/karelvanhecke/libvirt-operator/commit/252f8f5e1510d3ce344ae5add371a2f6b79b9969))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.19.1 ([#8](https://github.com/karelvanhecke/libvirt-operator/issues/8)) ([3a2209f](https://github.com/karelvanhecke/libvirt-operator/commit/3a2209f1b38e0f339541ba6029c1399be411265c))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.19.2 ([#74](https://github.com/karelvanhecke/libvirt-operator/issues/74)) ([3193b20](https://github.com/karelvanhecke/libvirt-operator/commit/3193b2037b73cd84cb8c314eb068b25236aca8af))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.19.3 ([#85](https://github.com/karelvanhecke/libvirt-operator/issues/85)) ([fb506fc](https://github.com/karelvanhecke/libvirt-operator/commit/fb506fcd1cd709c2247d403e662656f07414e3ba))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.19.4 ([#161](https://github.com/karelvanhecke/libvirt-operator/issues/161)) ([4a30d0c](https://github.com/karelvanhecke/libvirt-operator/commit/4a30d0cec6eac215f23cc9a7e80bd5717ab1c09d))
* **store/auth:** prevent possible nil dereference ([#167](https://github.com/karelvanhecke/libvirt-operator/issues/167)) ([9c9555f](https://github.com/karelvanhecke/libvirt-operator/commit/9c9555fb51ca8713f6932c7ad89ff4404a4df1a6))
* **store/host:** fix concurrent write error to session map ([#159](https://github.com/karelvanhecke/libvirt-operator/issues/159)) ([18cf460](https://github.com/karelvanhecke/libvirt-operator/commit/18cf46071eba74611fd0ccee5cf3367439816979))
* **store:** data wasn't cleaned up for deregistered auth entries ([#99](https://github.com/karelvanhecke/libvirt-operator/issues/99)) ([87a9190](https://github.com/karelvanhecke/libvirt-operator/commit/87a9190e6c43788cb73196a9105b1fe56fff0073))
* **util:** add missing unit to ConvertToBytes ([#157](https://github.com/karelvanhecke/libvirt-operator/issues/157)) ([703773e](https://github.com/karelvanhecke/libvirt-operator/commit/703773e4691658961682197a13b556ec4db9e317))


### Miscellaneous Chores

* set initial version v0.1.0 ([#6](https://github.com/karelvanhecke/libvirt-operator/issues/6)) ([df46027](https://github.com/karelvanhecke/libvirt-operator/commit/df46027efff11e556172e866d3a18507bc9e856e))
