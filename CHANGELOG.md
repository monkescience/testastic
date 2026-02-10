# Changelog

## [0.1.0](https://github.com/monkescience/testastic/compare/v0.0.2...v0.1.0) (2026-02-10)


### ⚠ BREAKING CHANGES

* migrate to unified functional options pattern

### Features

* add ErrorAs, Panics, NotPanics, MapHasValue assertions and fix ErrorIs nil target ([52b921c](https://github.com/monkescience/testastic/commit/52b921c4e6718acd13773f79c9c422c65eeb4e60))
* validate options per assertion type and consolidate test mocks ([ed09a98](https://github.com/monkescience/testastic/commit/ed09a98e3ddc237222ff7f88c40fd0f31d2743b3))


### Bug Fixes

* **file:** use plain text file creator instead of JSON parser in AssertFile update mode ([507545f](https://github.com/monkescience/testastic/commit/507545f1943f0bb99f75667a778c2b4ebd9536b5))
* **update:** only replace matcher at target JSON path, not all duplicate keys ([3c408e9](https://github.com/monkescience/testastic/commit/3c408e94979946e944edfced84547c76cbcc2a03))
* **yaml:** fix double-brace wrapping and strip YAML quotes from matcher expressions on update ([1461ab5](https://github.com/monkescience/testastic/commit/1461ab5df954236e8f67ff5cbf57d57260f0092a))


### Code Refactoring

* migrate to unified functional options pattern ([4ad65af](https://github.com/monkescience/testastic/commit/4ad65afce9218a0d598097b8c91e3cf350555892))

## [0.0.2](https://github.com/monkescience/testastic/compare/v0.0.1...v0.0.2) (2026-02-01)


### Features

* add AssertFile function with matcher support ([485dbcd](https://github.com/monkescience/testastic/commit/485dbcd926ca685bf94e98530f930c03bb062b34))
* add basic compareFileLines for exact matching ([e3d8fe9](https://github.com/monkescience/testastic/commit/e3d8fe95277e350e9c3bb5e72ac3ea71f36525fa))
* add compareFileLinesWithMatchers with pattern matching ([2a22bc1](https://github.com/monkescience/testastic/commit/2a22bc18118573b14d893e045b23205f2ec2b9e9))
* add Eventually assertions for async testing ([c889acf](https://github.com/monkescience/testastic/commit/c889acf2f1306a354c0d6fe8962eb7909258cf80))
* add FileConfig type for file assertions ([dd69d54](https://github.com/monkescience/testastic/commit/dd69d54ac40ef76a7a8067f778a4e4ef31a43d37))
* add HTML comparison support with flexible configuration options ([b77258d](https://github.com/monkescience/testastic/commit/b77258d65048ba6ebd5e3b34472e56caf35aa702))
* add initial implementation ([e31958a](https://github.com/monkescience/testastic/commit/e31958ac466c24dba8e3835f93511e60cb133197))
* add inline diff output for file assertions ([0a3f1d1](https://github.com/monkescience/testastic/commit/0a3f1d1440dcb28fc5f3c6f484c399984e4617be))
* add lineMatcher type and basic parseLine ([9fcc1aa](https://github.com/monkescience/testastic/commit/9fcc1aad60929753d95f0bbb1adc9cd83079d668))
* add Renovate configuration and GitHub workflow for scheduled dependency updates ([7907ec7](https://github.com/monkescience/testastic/commit/7907ec764737a70df6d45e1af8239e3ec1c82384))
* extend HTML parsing with embedded matcher support and add comprehensive tests ([70d9487](https://github.com/monkescience/testastic/commit/70d94877922c22f621041a644af8a23f9e15640f))
* implement parseLine with matcher detection ([a0b12f1](https://github.com/monkescience/testastic/commit/a0b12f16974c97ec56a4395e88a1dd966a82d286))
* improve regex matcher to support nested braces and add test ([22bb945](https://github.com/monkescience/testastic/commit/22bb94593d6687b591ec454a838202afbf5799f2))


### Performance Improvements

* cache compiled regex in TemplateString ([c99e780](https://github.com/monkescience/testastic/commit/c99e780d5387e149f798d8206899bba8776673cb))
