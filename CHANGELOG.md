# Changelog

## [0.3.2](https://github.com/openkcm/checker/compare/v0.3.1...v0.3.2) (2026-05-18)


### Bug Fixes

* add coderabbit file ([#157](https://github.com/openkcm/checker/issues/157)) ([6e38f83](https://github.com/openkcm/checker/commit/6e38f8364cea0657134610fbe2bd65723d2a217c))
* add default tag name if was missed from configuration ([#158](https://github.com/openkcm/checker/issues/158)) ([124fe30](https://github.com/openkcm/checker/commit/124fe30f5cb7b26deb6aefd2eeeceba7266946a8))
* **deps:** bump go.opentelemetry.io/otel/metric from 1.40.0 to 1.42.0 ([#149](https://github.com/openkcm/checker/issues/149)) ([70d4fe5](https://github.com/openkcm/checker/commit/70d4fe5e85e90a65f8adeb1eb47a6cc77f7b9a3c))
* **deps:** bump go.opentelemetry.io/otel/trace from 1.40.0 to 1.42.0 ([#151](https://github.com/openkcm/checker/issues/151)) ([0f708d9](https://github.com/openkcm/checker/commit/0f708d91bc1efa5672150edc550521b2cb574ea6))
* **deps:** bump google.golang.org/grpc from 1.79.2 to 1.79.3 ([#159](https://github.com/openkcm/checker/issues/159)) ([9e53fe2](https://github.com/openkcm/checker/commit/9e53fe2e8e91c70c9dfebf3b591690021873da29))
* **deps:** bump linkerd2 from edge-26.1.2 to edge-26.5.2 ([#170](https://github.com/openkcm/checker/issues/170)) ([64057fd](https://github.com/openkcm/checker/commit/64057fd708ce9246ef669834eecf10bdd9e87817))
* **deps:** bump the gomod-group group with 2 updates ([#156](https://github.com/openkcm/checker/issues/156)) ([f63c728](https://github.com/openkcm/checker/commit/f63c728b066aa6661ab2498f27cef92d61150d2d))
* extract repeated error strings into named constants ([#171](https://github.com/openkcm/checker/issues/171)) ([94351f8](https://github.com/openkcm/checker/commit/94351f86ef642a3eb7336950c4313630612cd0d2))
* Update dependabot config ([#155](https://github.com/openkcm/checker/issues/155)) ([f1432a2](https://github.com/openkcm/checker/commit/f1432a2a52d61e100a38d676369d69fe8fb8f208))

## [0.3.1](https://github.com/openkcm/checker/compare/v0.3.0...v0.3.1) (2026-02-27)


### Bug Fixes

* include back the tag configuration ([#143](https://github.com/openkcm/checker/issues/143)) ([fe6cd69](https://github.com/openkcm/checker/commit/fe6cd6929d1b2828ae563c1bf2b781e512a28202))

## [0.3.0](https://github.com/openkcm/checker/compare/v0.2.5...v0.3.0) (2026-02-27)


### Features

* Add topology constraints to helm chart ([#138](https://github.com/openkcm/checker/issues/138)) ([6b1c1dc](https://github.com/openkcm/checker/commit/6b1c1dc19e857f1c041c94c17ba777839ffe1964))


### Bug Fixes

* add support for retry for failure tolleration ([#141](https://github.com/openkcm/checker/issues/141)) ([d67a529](https://github.com/openkcm/checker/commit/d67a529c5b5658ff85f778da1bd3a3f6fc090141))
* include the healthchecks configuration in the helm chart ([#139](https://github.com/openkcm/checker/issues/139)) ([efcba7f](https://github.com/openkcm/checker/commit/efcba7fd463a8dc7fb6009b44332e1701dda1b7b))

## [0.2.5](https://github.com/openkcm/checker/compare/v0.2.4...v0.2.5) (2026-02-25)


### Bug Fixes

* include a new endpoint to check health check ([#136](https://github.com/openkcm/checker/issues/136)) ([5644976](https://github.com/openkcm/checker/commit/5644976dee4d14476b0716acdb8e1271156f65f2))

## [0.2.4](https://github.com/openkcm/checker/compare/v0.2.3...v0.2.4) (2026-01-15)


### Bug Fixes

* **deps:** bump github.com/openkcm/common-sdk from 1.7.0 to 1.8.0 ([#114](https://github.com/openkcm/checker/issues/114)) ([e7adcac](https://github.com/openkcm/checker/commit/e7adcacdbaed6a82820349a8ad1498ae9f00bd2c))
* **deps:** Update dependencies ([#124](https://github.com/openkcm/checker/issues/124)) ([a2e7856](https://github.com/openkcm/checker/commit/a2e785600320a1d080bfe6bbf3b65387cf996ec3))
* Fix linter error ([#110](https://github.com/openkcm/checker/issues/110)) ([ad8bc7b](https://github.com/openkcm/checker/commit/ad8bc7b5f3819ee092562e4a75a4e5a122b0769c))
* Fix make test target as no integration tests exist yet ([#109](https://github.com/openkcm/checker/issues/109)) ([ce43dc7](https://github.com/openkcm/checker/commit/ce43dc7085b7ac8b5c034775a97c44ba8694caf2))
* Update deps to fix vulnerability with linkerd ([#112](https://github.com/openkcm/checker/issues/112)) ([bf97180](https://github.com/openkcm/checker/commit/bf97180c2737b11aade429e5019c2d9452fcf7ac))
* warning has been reported as issues ([#126](https://github.com/openkcm/checker/issues/126)) ([9be5120](https://github.com/openkcm/checker/commit/9be51201510e3cf3cb0b13c2e1003eabab802377))

## [0.2.3](https://github.com/openkcm/checker/compare/v0.2.2...v0.2.3) (2025-11-20)


### Bug Fixes

* **deps:** bump golang.org/x/crypto from 0.41.0 to 0.45.0 in the go_modules group across 1 directory ([#104](https://github.com/openkcm/checker/issues/104)) ([cbc8f85](https://github.com/openkcm/checker/commit/cbc8f8517e092ae8602795bb7f4e0dfcd541b018))
* improve the checker by decoding the value coming from services as base64 value ([#105](https://github.com/openkcm/checker/issues/105)) ([84b4a2d](https://github.com/openkcm/checker/commit/84b4a2df4873b70a59bfe76647ebc0986b179ede))

## [0.2.2](https://github.com/openkcm/checker/compare/v0.2.1...v0.2.2) (2025-11-18)


### Bug Fixes

* include the value received from the version endpoint in case of unable to decode it ([#102](https://github.com/openkcm/checker/issues/102)) ([0f5680b](https://github.com/openkcm/checker/commit/0f5680bb725c4e75ced2d6bc6f2cae6bd5f11c29))

## [0.2.1](https://github.com/openkcm/checker/compare/v0.2.0...v0.2.1) (2025-11-18)


### Bug Fixes

* on the k8s cluster the response are empty ([#100](https://github.com/openkcm/checker/issues/100)) ([fc3cb4d](https://github.com/openkcm/checker/commit/fc3cb4dbf6e737bacc40f819b0a636a49febfe0d))

## [0.2.0](https://github.com/openkcm/checker/compare/v0.1.4...v0.2.0) (2025-11-17)


### Features

* refactor the github actions ([#68](https://github.com/openkcm/checker/issues/68)) ([8236e47](https://github.com/openkcm/checker/commit/8236e475be247beeec5270452c27aefe155ab077))


### Bug Fixes

* add versions in configmap ([#94](https://github.com/openkcm/checker/issues/94)) ([d13f300](https://github.com/openkcm/checker/commit/d13f30098b0244abf6cdeebe84760be10fbac0c6))
* adjust to write versions into the response ([#97](https://github.com/openkcm/checker/issues/97)) ([23793b5](https://github.com/openkcm/checker/commit/23793b5da95bd73b416fd08e0ef9a6d9c2ecf68f))
* **deps:** bump github.com/openkcm/common-sdk from 1.4.0 to 1.6.0 ([#92](https://github.com/openkcm/checker/issues/92)) ([f71adf9](https://github.com/openkcm/checker/commit/f71adf9dd89902b8377d126fcef14df5e72e3935))
* **deps:** bump github.com/samber/oops from 1.19.3 to 1.19.4 ([#96](https://github.com/openkcm/checker/issues/96)) ([25fcd96](https://github.com/openkcm/checker/commit/25fcd96c8f4f19acd3f0e6c841716cc4be7c1696))
* **deps:** bump helm.sh/helm/v3 from 3.17.4 to 3.18.5 in the go_modules group across 1 directory ([#95](https://github.com/openkcm/checker/issues/95)) ([9133ee8](https://github.com/openkcm/checker/commit/9133ee8fbf2bb7a4d4041053b6384d8cd5d39889))
* update the build info ([#81](https://github.com/openkcm/checker/issues/81)) ([2b514fc](https://github.com/openkcm/checker/commit/2b514fcc985cdade4e3cc6d945bdff4b8b1ec32c))
