# Changelog

## [1.6.2](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.6.1...v1.6.2) (2025-01-17)


### Bug Fixes

* **examples:** do not manually set env vars in deployment ([#653](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/653)) ([4a2d655](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/4a2d65544d47f1910c1a6501b5204cca77ba3548))

## [1.6.1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.6.0...v1.6.1) (2024-12-12)


### Bug Fixes

* Update Auth Proxy to version v2.14.2 ([#650](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/650)) ([35526d4](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/35526d46dfdb94d1b5b919a87adb3a7948bebfe6))
* Update dependencies. ([#648](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/648)) ([555c117](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/555c117103e0c74fee77ce59cae509d34b05b7a6))

## [1.6.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.5.1...v1.6.0) (2024-11-22)


### Features

* Add --min-sigterm-delay property to the workload configuration ([#639](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/639)) ([b4c226a](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/b4c226acad2d0af9860eb191da96637f6906f94e)), closes [#627](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/627)
* Run the proxy container as Sidecar Init Container ([#624](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/624)) ([19d8043](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/19d8043c3a7a2368ce797b3d7766656002cf5c6f)), closes [#381](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/381)


### Bug Fixes

* Update proxy version to v2.14.1 ([#632](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/632)) ([85655e5](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/85655e517e2080550ded7c830be29c4b8256064f))

## [1.5.1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.5.0...v1.5.1) (2024-08-22)


### Bug Fixes

* Add seconds unit to CSQL_PROXY_MAX_SIGTERM_DELAY value. ([#611](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/611)) ([c4eb455](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/c4eb4550e2423d716baf7e3d380894ac0b917601)), closes [#610](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/610)

## [1.5.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.4.5...v1.5.0) (2024-07-09)


### Features

* Add refresh strategy proxy configuration. Part of [#597](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/597). ([#598](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/598)) ([591c2a8](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/591c2a8763b8aac2ca0d9a21b727dfa79bc84575))


### Bug Fixes

* Bump dependencies to latest. ([#590](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/590)) ([f79adb0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/f79adb076b7bb68a727c76b3b850be031cf7df65))

## [1.4.5](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.4.4...v1.4.5) (2024-04-18)


### Miscellaneous Chores

* Trigger release v1.4.5 ([#580](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/580)) ([95978d3](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/95978d3712318ad5adc38574e945588a6bbbdb64))

## [1.4.4](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.4.3...v1.4.4) (2024-03-14)


### Bug Fixes

* update dependencies to latest versions ([#554](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/554)) ([51cbbd2](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/51cbbd2b9d8b3b2926551dbaf1a76c8aa016a69b))

## [1.4.3](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.4.2...v1.4.3) (2024-02-21)


### Bug Fixes

* update dependencies to latest versions ([#534](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/534)) ([3dedd86](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/3dedd860fdf1c6c7ba9079463ddcb4b0b3999cd6))
* update dependencies to latest versions ([#537](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/537)) ([7a1fc9c](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/7a1fc9cc75a7cbddd7c2b88da809f577516b2bb8))

## [1.4.2](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.4.1...v1.4.2) (2024-01-25)


### Bug Fixes

* update dependencies to latest versions ([#513](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/513)) ([0de141c](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0de141c3c88e99508c339cf377df74613bf818f8))

## [1.4.1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.4.0...v1.4.1) (2023-12-12)


### Bug Fixes

* configure webhook to ignore kube-system ([#499](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/499)) ([291de15](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/291de1535f12a727a78607d6603901209eaf8763))

## [1.4.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.3.0...v1.4.0) (2023-11-14)


### Features

* Add support for Service Account Impersonation. ([#445](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/445)) ([4d8e277](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/4d8e2777dd5ab2bcbba06cfdcc3a3320bea91a46)), closes [#392](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/392)
* Allow users to set quiet logging on proxy container ([#464](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/464)) ([1eaf019](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/1eaf019f3c2871dfd0e72b3eacd712712f7838ca)), closes [#402](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/402)

## [1.3.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.2.0...v1.3.0) (2023-10-17)


### Features

* Configure containerPort on proxy pod when telemetry is enabled ([#442](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/442)) ([a13ca22](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/a13ca224737d845b3faa8b7b066475d59ad9ddb9))


### Bug Fixes

* Correct the name of the quit url envvar CSQL_PROXY_QUIT_URLS. ([#454](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/454)) ([bd75451](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/bd75451235366725642894f454a3ce588bc4422d))

## [1.2.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.1.0...v1.2.0) (2023-09-20)


### Features

* Configure proxy container for graceful termination. ([#425](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/425)) ([0e0bb40](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0e0bb40339f5d4ea2659f81cc4f55ea15a2e3938)), closes [#361](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/361)


### Bug Fixes

* IAM api must be enabled before it is used ([#429](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/429)) ([0764b8b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0764b8b71b0c8ad0f5d97753b0587039bd7c47a9))

## [1.1.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.0.2...v1.1.0) (2023-07-20)


### Features

* Add support for Private Service Connect ([#391](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/391)) ([116b776](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/116b776ef63defe6a1e764ad6de46cb03d641c45)), closes [#389](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/389)

## [1.0.2](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.0.1...v1.0.2) (2023-07-06)


### Bug Fixes

* Case issue with pod State ([#386](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/386)) ([#387](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/387)) ([0a45c32](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0a45c32fad3f6b74922d23858a5b883d31d57368))

## [1.0.1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v1.0.0...v1.0.1) (2023-06-14)


### Miscellaneous Chores

* Use Kubernetes 1.27 compatible libraries. [#380](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/380) ([d4a21e](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/d4a21e67de2b3813f913ecc7eb37ef766b4f2e56))
* Prepare release 1.0.1 ([#382](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/382)) ([216b6d1](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/216b6d185064e1d89ad4010a17ffc8f09777a026))


## [1.0.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.5.0...v1.0.0) (2023-05-16)


### Bug Fixes

* delete misconfigured pods in error state. ([#338](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/338)) ([4a02aa7](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/4a02aa72bf4ace836b6c8af345302617b7f90765)), closes [#337](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/337)


### Miscellaneous Chores

* Prepare release 1.0.0 ([#353](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/353)) ([02506ff](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/02506ff275abe3d37d6f59c1c059da41a66cdfa0))

## [0.5.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.4.0...v0.5.0) (2023-04-26)


### Features

* Improve security posture of proxy containers. ([#322](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/322)) ([dc8911e](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/dc8911e3a0db5e32bc4c611ddbdcb875dbcc51e3))
* Make proxy container healthchecks more resilient. ([#321](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/321)) ([548a922](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/548a9222d54b6c85dd45dc5983beebac5e7f08ef))


### Bug Fixes

* The e2e k8s node pool version should match the master version ([#319](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/319)) ([fbfa004](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/fbfa00444da0f310f52c61ed4665a5153bebe2c4))


### Miscellaneous Chores

* Prepare release 0.5.0 ([#327](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/327)) ([5aeb27b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/5aeb27b100ca186bf74c4952a9252718cd43b60b))

## [0.4.0](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/compare/v0.3.0...v0.4.0) (2023-03-28)


### ⚠ BREAKING CHANGES

* Move to v1 for the AuthProxyWorkload api version. ([#258](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/258))

### Features

* Add configuration for the admin api port and debug. ([#213](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/213)) ([0ddd681](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/0ddd681407f2c2e2e8a6932e90ffda92ddc298c6))
* Automatically update default images on operator startup. ([#254](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/254)) ([2453be6](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/2453be626a30eef930ca136c941b6a3cbb9cbe99)), closes [#87](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/87)
* Configure --disable-metrics and --disable-telemetry flags. ([#222](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/222)) ([5be6c3b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/5be6c3b5a15f0bba773b9f007d855581d84284b8))
* Configure --quota-project flag on the proxy. ([#225](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/225)) ([c3b4f1b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/c3b4f1bac3b958c40cc9af2816a017c75c8eb006)), closes [#45](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/45)
* Configure Google Cloud Telemetry flags on the proxy. ([#223](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/223)) ([76b0f39](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/76b0f39b4b9a677446fd3a3948e345bfaad1c432))
* Configure prometheus flags on the proxy. ([#224](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/224)) ([a055d3b](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/a055d3b79de5ac54ecf2af82b2bbdebcb1551307))
* Move to v1 for the AuthProxyWorkload api version. ([#258](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/258)) ([7b65d5c](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/7b65d5cbb6ca3279d0dd71c95b439104a1e5b8ca))
* Updating the RolloutStrategy field is not allowed. ([#212](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/212)) ([f31b637](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/f31b637373b073357431aec0d1b4176507e9a00c))
* Validate instance fields ([#221](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/221)) ([d516cc2](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/d516cc2661b3854a69e267daafe68f4c1c3b73ad)), closes [#36](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/36)
* When the operator's default proxy image changes, workload containers should be updated. ([#253](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/253)) ([220c855](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/220c8555d42fbc4f82e029ce1edd46c7a92648af))


### Bug Fixes

* Only process owner references for known kinds of owners. ([#245](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/245)) ([12be1dc](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/12be1dc2d6bf40f987200dfdec761e7b121b00c1)), closes [#244](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/244)
* Repair a bad merge of tool versions in the Makefile. ([#249](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/249)) ([f2ba903](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/f2ba903f91956ed4b2b923c7adcb846ca9162a26))
* Validate the AdminServer.EnableAPIs[] field properly. ([#263](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/issues/263)) ([115ac32](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/115ac329a10cf445ccf59c89e965db6c92ce5831))


### Miscellaneous Chores

* Prepare release 0.4.0 ([2e6a6ad](https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/commit/2e6a6adcd92dce926e4ccaa1f15d2386b5b30721))

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
