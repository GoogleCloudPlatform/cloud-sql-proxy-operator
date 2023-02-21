# Changelog

## [0.3.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.2.0...v0.3.0) (2023-02-21)


### Features

* add new field RolloutStrategy control automatic rollout ([#202](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/202)) ([090b88d](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/090b88da2f3cbc00ca98bee7cdfbb4e50a6c4cb9))
* Add new terraform project for e2e test resources ([#181](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/181)) ([0140592](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0140592b3a19087cc5ee769b542ae461f3a5d1b4))
* add script to run terraform with input validation. ([#182](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/182)) ([857444a](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/857444ac09b8c1c5c9c3536ed1cab7367f778015))
* Add support for Unix sockets. ([#205](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/205)) ([8177a35](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/8177a35be7988a01de682d806c05b9306537c3a1)), closes [#47](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/47)
* Add telemetry settings to configure health check port ([#210](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/210)) ([3ede42d](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/3ede42da9f502090d80b95970296f138484ef522))
* add the e2e test job for Cloud Build ([#184](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/184)) ([dc2990c](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/dc2990c4483d216a31a6cafbf45ebba6936b8c6a))
* automatic  changes to workloads when an AuthProxyWorload is deleted ([#200](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/200)) ([e11caed](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/e11caed179f82ca3d24322d9f80a95174911bddd))
* Automatically trigger pod rollout for appsv1 resources when AuthProxyWorkload changes. ([#197](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/197)) ([3b0359b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/3b0359b68b8d5c0dcd3e306102945c6e608ff095))
* separate terraform for project setup and permissions  ([#179](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/179)) ([8f43657](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/8f43657a6e039db0e3c8c57be56ec8d68ee503e9))
* Validate AuthProxyWorkload spec.selector field ([#209](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/209)) ([98c460b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/98c460bdd34dfa00815e664f60e38aa7327d92d4))
* Validate AuthProxyWorkload updates to prevent changes to the workload selector. ([#211](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/211)) ([4304283](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/4304283c1e85b079aab5cbf6c4c2dafb73ed654a))


### Miscellaneous Chores

* Release 0.3.0 ([5204cca](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/5204cca618b6bb7588da302d82c6735389eb700f))

## [0.2.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.1.0...v0.2.0) (2023-01-18)


### ⚠ BREAKING CHANGES

* remove Namespace field from AuthProxyWorkloadSelector ([#168](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/168))

### Bug Fixes

* remove Namespace field from AuthProxyWorkloadSelector ([#168](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/168)) ([7bcc27d](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/7bcc27d080f0f848da80740a2e4bbe75c0397031))
* Update installer.sh to use helm for cert-manager ([#163](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/163)) ([62fc5dc](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/62fc5dc49a7e373fc468a512c5e54f6adfcedde4)), closes [#157](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/157)


### Miscellaneous Chores

* release 0.2.0 ([#175](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/175)) ([44babcd](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/44babcd3dbe703f55b9bc464597a79bdf6adb718))

## [0.1.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.0.3...v0.1.0) (2022-12-13)


### Features

* add user agent to proxy invocation ([#122](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/122)) ([803446d](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/803446d4766fe556cb149725100f7e955bd8c8d0))


### Bug Fixes

* change memory resource to match recommendations Cloud SQL Proxy ([#139](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/139)) ([a475dd9](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/a475dd934a59469e9ef38fd9934593d7d7c3b0e6))
* remove unsupported CRD fields and associated code from the project. ([#141](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/141)) ([3867621](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/386762120f386a459c57c6e3e090e6795f53886f))

## [0.0.3](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.0.1...v0.0.3) (2022-12-07)


### Features

* Add a release job to generate code on release PRs GH-66 ([#110](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/110)) ([d23a484](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/d23a484e39376ca15468e119b726476bd49aca59))
* Add build and test targets to new makefile ([#96](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/96)) ([f6c3de3](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/f6c3de3189c563cd48228612133ef9f8a44e4f1f))
* add cert-manager deployment for e2e test environments. ([#28](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/28)) ([99a3104](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/99a3104980c73323a61c75f1b45e30ddeb3e4031))
* add cloud-build job for the release process GH-66 ([#108](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/108)) ([614067b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/614067ba3cb697e3672fd65bbb7df701817553b2))
* add data structure for the AuthProxyWorkload custom resource. ([#20](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/20)) ([2af04ad](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/2af04ad86458b6101f433b7a3e55647a710cc781))
* add e2e tests ([#40](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/40)) ([fd69001](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/fd69001b457f9378dea5af2f3c3dd8d0e2687c27))
* Add internal libraries for kubernetes names ([#23](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/23)) ([ee1e649](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/ee1e64979573c574b28d65d4deaf96a556e650b4))
* add lint target to makefile ([#101](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/101)) ([9f3d81b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/9f3d81bc2075722614c7c21c455940a54045f2b1))
* add pod webhook boilerplate ([#41](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/41) step 4) ([#115](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/115)) ([9d3971c](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/9d3971c55fb968a6f4e941b36595f3514f30ef15))
* Add reconcile controller logic issue [#37](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/37)  ([#39](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/39)) ([0be2c0d](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0be2c0d4ad596de025d940e2d04e6dd9087057a3))
* add webhook and test to see that it updates a new workload ([#34](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/34)) ([c01c5c6](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/c01c5c6263748e6163dc268928159f6b379b49e2))
* Adds Workload interface to access PodSpec for all workload types  ([#25](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/25)) ([c6706ec](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/c6706ec0e0cfa2ca71e5d84d848fecb9ef49ac42))
* AuthProxyWorkload Reconcile no longer modifies workloads ([#41](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/41) step 6) ([#117](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/117)) ([cfffe91](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/cfffe91764da38a5b91104b71798ae9863cb4144))
* commit installer scripts into repo as part of generate, GH-66 ([#107](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/107)) ([0dee3a1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0dee3a18cc230ae0f525fef2c19c1a977fe0ffe1))
* deploy to a kubernetes cluster defined in envvar KUBECONFIG ([#97](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/97)) ([13ba287](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/13ba28737d772d281a8823d4fafd0bc2c36fe970))
* Finish pod webhook and workload reconcile implementation ([#41](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/41) step 8) ([#119](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/119)) ([3be65d5](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/3be65d5e588c0ebfed7c01e4c4d1ddc2349f3e66))
* Logic to apply AuthProxyWorkload spec to workload containers ([#26](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/26)) ([17b73c0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/17b73c042e78fda422c9af0de1a0fb62e1c5f451))
* PodSpec changes now are applied only to pods on creation ([#41](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/41) step 7) ([#118](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/118)) ([df1f322](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/df1f32242a0800c8eb762daccc636a9d9eeded4e))
* Run terraform scripts to provision infrastructure ([#98](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/98)) ([2b572f3](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/2b572f304fdd7851f82899660d672e587a2e1e9d))
* Set the hardcoded default proxy image URL to a real url. gh-93 ([#94](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/94)) ([082267c](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/082267cad85b30cd1874ccf34a8564e611a6b05e))


### Bug Fixes

* add -race flag to go test ([#99](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/99)) ([11dce11](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/11dce114097f66e8ec0f7ed42a8c95c9aa01da95))
* correct paths in install.sh installer script template [#66](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/66) ([#111](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/111)) ([2614115](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/2614115eefd508dab8656f6e7ea1b99f2015d02b))
* e2e_cleanup_test_namespaces succeeds when there are no matching namespaces  ([#120](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/120)) ([d4cabcd](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/d4cabcd1b91247b031a02530a79c5bffb14926c6))
* host should be '127.0.0.1' not 'localhost' ([#128](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/128)) ([77293e7](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/77293e73aafde77b57ba134ae0fb6597d16800c4)), closes [#124](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/124)
* Improve code and repair edge case problems with the internal PodSpec updates. ([#31](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/31)) ([fe6ce99](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/fe6ce99630da3d4858fdbe0f2e5f617d342df722))
* rename end to end test targets from `gcloud_*` to `e2e_*` for clarity ([#84](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/84)) ([a1fd817](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/a1fd81766161b8c72a77465a6889d05c5d54c773))
* timeout for golangci-lint is 5m. ([773c694](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/773c694355410325000ef0d3adc3a6c0d3581214))


### Miscellaneous Chores

* release 0.0.1 test release process ([4ab7f5b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/4ab7f5b7932157142a5446cee0fd44c49d379045))
* release 0.0.2 ([f97491a](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/f97491a3785f82721d1ccb3441049276ce6725ea))
* release 0.0.3 ([f1e10b5](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/f1e10b54c2689b607ecf67cabe0d766809077aaf))

## 0.0.1 (2022-10-12)


### Features

* add cert-manager deployment for e2e test environments. ([#28](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/28)) ([99a3104](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/99a3104980c73323a61c75f1b45e30ddeb3e4031))
* add data structure for the AuthProxyWorkload custom resource. ([#20](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/20)) ([2af04ad](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/2af04ad86458b6101f433b7a3e55647a710cc781))
* add e2e tests ([#40](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/40)) ([fd69001](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/fd69001b457f9378dea5af2f3c3dd8d0e2687c27))
* Add internal libraries for kubernetes names ([#23](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/23)) ([ee1e649](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/ee1e64979573c574b28d65d4deaf96a556e650b4))
* Add reconcile controller logic issue [#37](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/37)  ([#39](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/39)) ([0be2c0d](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0be2c0d4ad596de025d940e2d04e6dd9087057a3))
* add webhook and test to see that it updates a new workload ([#34](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/34)) ([c01c5c6](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/c01c5c6263748e6163dc268928159f6b379b49e2))
* Adds Workload interface to access PodSpec for all workload types  ([#25](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/25)) ([c6706ec](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/c6706ec0e0cfa2ca71e5d84d848fecb9ef49ac42))
* Logic to apply AuthProxyWorkload spec to workload containers ([#26](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/26)) ([17b73c0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/17b73c042e78fda422c9af0de1a0fb62e1c5f451))


### Bug Fixes

* Improve code and repair edge case problems with the internal PodSpec updates. ([#31](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/31)) ([fe6ce99](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/fe6ce99630da3d4858fdbe0f2e5f617d342df722))


### Miscellaneous Chores

* release 0.0.1 test release process ([4ab7f5b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/4ab7f5b7932157142a5446cee0fd44c49d379045))
