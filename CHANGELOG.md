# Changelog

## [v0.4.6](https://github.com/monkescience/testastic/compare/v0.4.5...v0.4.6) (2026-09-03)

### Bug Fixes

- bound process output capture ([2528163](https://github.com/monkescience/testastic/commit/2528163e4b276c0f640505be56625f1a164d72e6))
- bound binary run output ([9e15357](https://github.com/monkescience/testastic/commit/9e15357835aadc9e72a030eeee9a173bac6de671))
- avoid cubic unordered matching ([fd71fef](https://github.com/monkescience/testastic/commit/fd71fefd4b991e682b7c5461ff281feb389db27a))
- prevent fixture environment disclosure ([9baa399](https://github.com/monkescience/testastic/commit/9baa399827798d144b598cecf9ccf0bda5933f94))

## [v0.4.5](https://github.com/monkescience/testastic/compare/v0.4.4...v0.4.5) (2026-08-28)

### Bug Fixes

- remove obsolete matcher code ([b009ad1](https://github.com/monkescience/testastic/commit/b009ad12d3f6b215e0e22d615620526a28aab1a2))

## [v0.4.4](https://github.com/monkescience/testastic/compare/v0.4.3...v0.4.4) (2026-08-25)

### Bug Fixes

- stabilize Windows platform tests ([c33b784](https://github.com/monkescience/testastic/commit/c33b784f3a18e3bba95bdaeb45889750b0f7fdc6))
- preserve YAML streams and unordered matches ([770120b](https://github.com/monkescience/testastic/commit/770120b14512004d8653d8c30860dc130fb80c71))
- compare YAML document streams ([bf04c84](https://github.com/monkescience/testastic/commit/bf04c842d533b55a4a2f6c01839077249cab43c6))
- honor unordered matches in document diffs ([384803b](https://github.com/monkescience/testastic/commit/384803b2334d23ed31063191cd347e4bd5c51fc2))
- preserve JSON and YAML matcher literals ([365397f](https://github.com/monkescience/testastic/commit/365397ff140b8eaadb382d66fe4e96cb25008c1e))
- omit ignored comments from HTML diffs ([70f758b](https://github.com/monkescience/testastic/commit/70f758b11d342bc967f1b6287a549b34d2f71d50))
- retain HTML whitespace text nodes ([8153c4a](https://github.com/monkescience/testastic/commit/8153c4ae56b771b6d5ece9ec00bbfce6f637e1ad))
- avoid HTML matcher placeholder collisions ([d275967](https://github.com/monkescience/testastic/commit/d275967befffcbdce1fbdbb3f0e8abc251c4bf16))
- isolate file matcher captures ([d160efb](https://github.com/monkescience/testastic/commit/d160efbb81a224a483647933647b3ebe938f82d5))
- validate AnyDateTime semantics ([63a5ee2](https://github.com/monkescience/testastic/commit/63a5ee2a764ebb7c892ff97429093f6f84a807fa))
- bound Eventually callbacks by timeout ([98cd5be](https://github.com/monkescience/testastic/commit/98cd5bed8d56cc41d2f0c8670e0fd64caf7b1123))

## [v0.4.3](https://github.com/monkescience/testastic/compare/v0.4.2...v0.4.3) (2026-08-23)

### Bug Fixes

- bound diff matrix allocation ([502f4bc](https://github.com/monkescience/testastic/commit/502f4bc97e5c3ad63bdd6efa8729824cb2c7ab04))
- bound blocking readiness checks by timeout ([5c37f62](https://github.com/monkescience/testastic/commit/5c37f62106c2f8763ed16d6426acdf95f6339630))
- bound inherited output wait after run timeout ([1ba0c06](https://github.com/monkescience/testastic/commit/1ba0c065c3fda743a675938f5cc881dd1704af08))
- prevent diff allocation size overflow ([e54e3a6](https://github.com/monkescience/testastic/commit/e54e3a62c9b177872525f1ec51142c5948ee1830))

## [v0.4.2](https://github.com/monkescience/testastic/compare/v0.4.1...v0.4.2) (2026-08-23)

### Bug Fixes

- compare working directories by identity ([94d58f7](https://github.com/monkescience/testastic/commit/94d58f7f0d941b5d73f23fab894af8b55342ac9c))
- **deps:** update module golang.org/x/net to v0.58.0 (#90) ([354c73e](https://github.com/monkescience/testastic/commit/354c73e5821eb8956a50c2e21b6ba9d90405db41))

## [v0.4.1](https://github.com/monkescience/testastic/compare/v0.4.0...v0.4.1) (2026-07-18)

### Bug Fixes

- **deps:** update module golang.org/x/net to v0.57.0 (#72) ([94af2a6](https://github.com/monkescience/testastic/commit/94af2a6f555ce291e6fab11c515c2b347f93365a))
- **deps:** update module golang.org/x/term to v0.45.0 (#73) ([b01da7e](https://github.com/monkescience/testastic/commit/b01da7e97b7c9f3667b9d2cc9b3f51f198cfd4c3))
- **deps:** update module golang.org/x/net to v0.56.0 (#61) ([5aa27ed](https://github.com/monkescience/testastic/commit/5aa27edfd61062f1382a2fe64d98b313aaf79e84))

## [v0.4.0](https://github.com/monkescience/testastic/compare/v0.3.5...v0.4.0) (2026-06-06)

### ⚠ BREAKING CHANGES

- correct comparison, matcher, and golden-file engine behavior ([4b5ef76](https://github.com/monkescience/testastic/commit/4b5ef76e4ebde523364cb03b360c28d86ea09697))
- **assert:** fail on NaN ordering and make caller misuse fatal ([5188285](https://github.com/monkescience/testastic/commit/518828545f59c4bfe463ba14eb983b9bb83033b2))
### Bug Fixes

- correct comparison, matcher, and golden-file engine behavior ([4b5ef76](https://github.com/monkescience/testastic/commit/4b5ef76e4ebde523364cb03b360c28d86ea09697))
- **binary:** preserve exit code on cancel, clean temp dir on build failure ([46eb57b](https://github.com/monkescience/testastic/commit/46eb57b9e02d61d9ccf4a6663ae7d82c5b1c47a7))
- **process:** release context on start failure, fix Windows shutdown doc ([3c9dd3e](https://github.com/monkescience/testastic/commit/3c9dd3e7db4e800e3f09093c3012142a735c5a9e))
- **assert:** fail on NaN ordering and make caller misuse fatal ([5188285](https://github.com/monkescience/testastic/commit/518828545f59c4bfe463ba14eb983b9bb83033b2))
- **eventually:** evaluate at the deadline and bound the timeout from entry ([3bee6c9](https://github.com/monkescience/testastic/commit/3bee6c94133f0e35b8b30fbf3d15d2cd2efb320a))
- **deps:** update module golang.org/x/net to v0.55.0 (#56) ([aad7548](https://github.com/monkescience/testastic/commit/aad75481f9043fb3eccf87a119cd87df1230b407))
- **deps:** update module golang.org/x/net to v0.54.0 (#53) ([a39356a](https://github.com/monkescience/testastic/commit/a39356a0af8353ecbaf5674aa28ef718e4479e44))

## [v0.3.5](https://github.com/monkescience/testastic/compare/v0.3.4...v0.3.5) (2026-05-14)

### Bug Fixes

- **coverage:** isolate each run in its own GOCOVERDIR subdir ([8fb67e0](https://github.com/monkescience/testastic/commit/8fb67e0551fad113f1ab14f1fe84b61365cf4980))

## [v0.3.4](https://github.com/monkescience/testastic/compare/v0.3.3...v0.3.4) (2026-04-27)

### Bug Fixes

- **html:** handle typed matchers and escaping ([d6b2ddb](https://github.com/monkescience/testastic/commit/d6b2ddb266621f532d4aaec8e7f0bcc96d43688b))

## [v0.3.3](https://github.com/monkescience/testastic/compare/v0.3.2...v0.3.3) (2026-04-26)

### Bug Fixes

- **coverage:** fail on conversion errors ([44fda87](https://github.com/monkescience/testastic/commit/44fda87cf44b5380721c5c75666cd5f877d982ee))
- **assert:** enforce eventually deadline ([f0901a0](https://github.com/monkescience/testastic/commit/f0901a0721ce2d5e7ecc41788c9263a17d5b2894))
- **json:** preserve update number precision ([a7fea9b](https://github.com/monkescience/testastic/commit/a7fea9bf2a60c54466dcb54f5714be0ce5f09f31))
- **json:** reject trailing json content ([0a68d22](https://github.com/monkescience/testastic/commit/0a68d22df39415e5df7ae6c5e90e90b80e9d22f4))

## [v0.3.2](https://github.com/monkescience/testastic/compare/v0.3.1...v0.3.2) (2026-04-26)

### Bug Fixes

- **color:** guard useColors cache against concurrent access ([733b922](https://github.com/monkescience/testastic/commit/733b9221e438d2c5e73b299f4c39871d3a462b2d))

## [v0.3.1](https://github.com/monkescience/testastic/compare/v0.3.0...v0.3.1) (2026-04-26)

### Bug Fixes

- **compare:** preserve numeric precision ([b89d1ac](https://github.com/monkescience/testastic/commit/b89d1ac1a78a515db307fc4403cfb58e07e7dcb6))
- **compare:** backtrack unordered matches ([caba243](https://github.com/monkescience/testastic/commit/caba243976552d88e24206611deaf4c05a0a35cf))
- **process:** stop process when readiness fails ([bb483f1](https://github.com/monkescience/testastic/commit/bb483f16768c1747e0e9fd28338ca2119790b831))

## [v0.3.0](https://github.com/monkescience/testastic/compare/v0.2.1...v0.3.0) (2026-04-16)

### ⚠ BREAKING CHANGES

- add Binary API for subprocess testing ([00d315e](https://github.com/monkescience/testastic/commit/00d315e9b3a12cf5744840319c041b47f3dda3d3))
### Features

- add Binary API for subprocess testing ([00d315e](https://github.com/monkescience/testastic/commit/00d315e9b3a12cf5744840319c041b47f3dda3d3))

## [v0.2.1](https://github.com/monkescience/testastic/compare/v0.2.0...v0.2.1) (2026-04-10)

### Bug Fixes

- **deps:** update module golang.org/x/net to v0.53.0 (#39) ([5e0e5d2](https://github.com/monkescience/testastic/commit/5e0e5d2e37135340e35d3663ba4ed4ffdfcfb0a6))
- **deps:** update module golang.org/x/term to v0.42.0 (#34) ([d36f4ba](https://github.com/monkescience/testastic/commit/d36f4ba3dde2ff547def8827fb5a7bdb173e90f4))
- **deps:** update module golang.org/x/net to v0.52.0 (#33) ([02d4fee](https://github.com/monkescience/testastic/commit/02d4fee82d3cc9a59bd2004ee8ba0b1f314f2b87))

## [v0.2.0](https://github.com/monkescience/testastic/compare/v0.1.3...v0.2.0) (2026-04-07)

### ⚠ BREAKING CHANGES

- replace ProcessConfig with functional options pattern ([b75586f](https://github.com/monkescience/testastic/commit/b75586f791c3bf9b7280ed041773fbbe77f3b5b2))
### Features

- add CollectProcessCoverage TestMain helper ([fcc9d51](https://github.com/monkescience/testastic/commit/fcc9d51ca1341527aca9047c74cf1b1fd21e5980))
- add blackbox service testing with coverage instrumentation ([a73e8dc](https://github.com/monkescience/testastic/commit/a73e8dcb67346bb7662e9af5d654f65b315a09e1))
### Bug Fixes

- cover missing test paths and honor HTML IgnoreFields ([13df2eb](https://github.com/monkescience/testastic/commit/13df2eb5b149013a344304a509fc63c0e71b26ad))
- resolve bugs and align tests with codebase conventions ([6ac3957](https://github.com/monkescience/testastic/commit/6ac39571f254592adf8738280b198a23a402fb7a))

## [0.1.3](https://github.com/monkescience/testastic/compare/v0.1.2...v0.1.3) (2026-03-22)


### Bug Fixes

* **assert:** keep matched AssertFile lines out of diffs ([e55f0d7](https://github.com/monkescience/testastic/commit/e55f0d7d2fb1952a0b5c63ebd56c84b49a8da4d7))
* **deps:** update module golang.org/x/net to v0.51.0 ([#21](https://github.com/monkescience/testastic/issues/21)) ([7595901](https://github.com/monkescience/testastic/commit/7595901404e7fa7850df6437e0f128851f6d9776))

## [0.1.2](https://github.com/monkescience/testastic/compare/v0.1.1...v0.1.2) (2026-03-06)


### Features

* **assert:** add negative prefix and suffix assertions ([21a2615](https://github.com/monkescience/testastic/commit/21a26155245560fa1c4049579ef0367182c612ee))
* **assert:** support string-like inputs and map value absence checks ([f389179](https://github.com/monkescience/testastic/commit/f3891795618e9c33e20aa7876a4b5dcaea3aa111))

## [0.1.1](https://github.com/monkescience/testastic/compare/v0.1.0...v0.1.1) (2026-02-11)


### Bug Fixes

* harden expected file writes and matcher registry ([535cebb](https://github.com/monkescience/testastic/commit/535cebbbc497d79219e9ba11a32c30a97592b317))

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
