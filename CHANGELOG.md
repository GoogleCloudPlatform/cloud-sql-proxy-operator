# Changelog

## [0.0.1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.0.1...v0.0.1) (2022-12-07)


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
