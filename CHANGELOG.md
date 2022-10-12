# Changelog

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
