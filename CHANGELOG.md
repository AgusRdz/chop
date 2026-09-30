# Changelog

All notable changes to chop are documented here.

## [1.39.2] - 2026-09-30

### Bug Fixes
- Avoid '<--' in init usage, which renders as an arrow in ligature fonts
([7f0313c](https://github.com/AgusRdz/chop/commit/7f0313c1d692d8b33aaef6ca566fa353cb4414aa))
- Only report conflicts from enabled plugins
([4d8fbd8](https://github.com/AgusRdz/chop/commit/4d8fbd853dc313ce044f771fd4f1fcdc2a8a0678))
- Consistent advice for plugin hook conflicts
([3d286e9](https://github.com/AgusRdz/chop/commit/3d286e9ab64fdd559990c86c7f783ab435ad71e8))
- Plain install confirmation message
([af9d543](https://github.com/AgusRdz/chop/commit/af9d543b7d8dc46970b4a1e0575a592a5a6a1b71))
- Plain install confirmation in install scripts
([2936bcf](https://github.com/AgusRdz/chop/commit/2936bcf58fad16ad6daa6f2231ad1fc021debb2f))

### Other
- Merge pull request #68 from AgusRdz/fix/init-and-hook-conflicts
([d832b56](https://github.com/AgusRdz/chop/commit/d832b56eb87769087f486439cc8b0511f08fc389))
## [1.39.1] - 2026-09-30

### Bug Fixes
- Drop download-and-execute one-liners from binary output
([c7dcac3](https://github.com/AgusRdz/chop/commit/c7dcac303f77b7e32e5a5f683af66b6ec5bf6cb4))
- Recommend dot-sourcing PowerShell completion instead of Invoke-Expression
([f6e4a96](https://github.com/AgusRdz/chop/commit/f6e4a9628b39b9254804472bf9a3928f826afa55))
- Plain output for init --agent-handshake
([783c24d](https://github.com/AgusRdz/chop/commit/783c24d39c278bed130e5d8a8aef1884c79c58ca))
- Drop detached self-spawn from wrapped commands; clearer staging; clean .old
([143b90f](https://github.com/AgusRdz/chop/commit/143b90ff234c8296a3e632da8b9eb17e0c0922a3))
- Report an available update
([9ae4727](https://github.com/AgusRdz/chop/commit/9ae472799a546a2ac99a47e30ecb436d2096d470))
- Run update housekeeping only from doctor and gain
([06aa5b3](https://github.com/AgusRdz/chop/commit/06aa5b3a6b36d0754f2543647ec938e4d4881294))

### CI/CD
- Authenticode-sign Windows release binary via SignPath
([aa68cef](https://github.com/AgusRdz/chop/commit/aa68cefc74be6d4a7999a7d96c4e422172a90609))

### Other
- Merge pull request #66 from AgusRdz/ci/windows-code-signing
([e6a6f6d](https://github.com/AgusRdz/chop/commit/e6a6f6d91fd0321f87d6cbeb721f71497136ffd6))
- Embed Windows version info and manifest via go-winres
([8a08e29](https://github.com/AgusRdz/chop/commit/8a08e29ea58a1ec876fb4c3c7617bb447985f077))
- Merge pull request #67 from AgusRdz/fix/av-heuristic-triggers
([bf966f5](https://github.com/AgusRdz/chop/commit/bf966f50b8a65f87266d660f7cf773ff84576245))
## [1.39.0] - 2026-09-13

### Features
- Add Pi coding agent support
([76a8da3](https://github.com/AgusRdz/chop/commit/76a8da38bbc212e04a92dda54ab986d85995f9e0))
## [1.38.14] - 2026-09-07

### Bug Fixes
- Preserve diff content instead of collapsing to bare stat counts
([f28381b](https://github.com/AgusRdz/chop/commit/f28381bcda45058d7bfffb4e4fbd7a02c3cc3a93))
## [1.38.13] - 2026-08-31

### Bug Fixes
- Match hook commands by executable
([8da10f2](https://github.com/AgusRdz/chop/commit/8da10f2b316094a66aea9c2ccc6b60269c4419e0))

### Other
- Merge pull request #65 from arkgum/fix/strict-hook-command-detection
([3930b85](https://github.com/AgusRdz/chop/commit/3930b85c4ac4af7a9cc8fdbf8a01592679020736))
## [1.38.12] - 2026-08-29

### Bug Fixes
- Honor global hook management flags (#63)
([bd5a0df](https://github.com/AgusRdz/chop/commit/bd5a0df5e76f325b417ad8b0720916cd936405c3))
- Preserve stable agent discovery paths (#64)
([7136b75](https://github.com/AgusRdz/chop/commit/7136b75ee4db0f9267a0864360a12ca46335ec77))
- Degrade gracefully when agent-info can't resolve executable path
([8275a60](https://github.com/AgusRdz/chop/commit/8275a60127d540fa2c1231656e82fd8273d9e519))
## [1.38.11] - 2026-08-27

### Bug Fixes
- Align doctor with stable hook paths (#62)
([0efffde](https://github.com/AgusRdz/chop/commit/0efffde55851cf17089424653b6ae8ab9892dc41))
## [1.38.10] - 2026-08-27

### Bug Fixes
- Preserve stable hook paths across package upgrades (#61)
([4b6b009](https://github.com/AgusRdz/chop/commit/4b6b0090157cf3f2ebe805ba97c6e88dd8de5f6b))
## [1.38.9] - 2026-08-26

### Bug Fixes
- Avoid false-positive streaming hook detection
([395e722](https://github.com/AgusRdz/chop/commit/395e722427ccf9a3384dedb134a145c49b8a90d6))
- Treat aws/argocd/flux log follow as streaming
([66b22af](https://github.com/AgusRdz/chop/commit/66b22af45ab84375b7f1e556c258bdc2a49ba490))

### Other
- Merge pull request #60 from arkgum/fix/hook-streaming-detection
([bc0830c](https://github.com/AgusRdz/chop/commit/bc0830c2e5a71ee8d3bf179be45d01a8404c9733))
## [1.38.8] - 2026-08-20

### Bug Fixes
- Route Maven and Gradle tasks after options (#59)
([613031f](https://github.com/AgusRdz/chop/commit/613031f642611c5b1c4c4687c37b47b841258005))
## [1.38.7] - 2026-06-15

### Bug Fixes
- Address all LOW findings from codebase code review
([1c498fc](https://github.com/AgusRdz/chop/commit/1c498fcd0211028f06138c750e1e3cdc47273366))
- Address all MEDIUM findings from codebase code review
([533cd4f](https://github.com/AgusRdz/chop/commit/533cd4f0463520b7a75f831ea1cbd2c73e056df9))
- Address all HIGH findings from codebase code review
([e3d508e](https://github.com/AgusRdz/chop/commit/e3d508e04330278003955639d53876ec3593e664))
- Correct SHA pin for actions/attest-build-provenance v2.3.0
([fc6c278](https://github.com/AgusRdz/chop/commit/fc6c2782dd9f67512af85b9908af162fa290d415))
## [1.38.6] - 2026-05-18

### Bug Fixes
- Strip trailing 2>&1 from wrapped commands
([0121d10](https://github.com/AgusRdz/chop/commit/0121d106a938a51e4fa3eeaaf066c1de6fbfad24))
## [1.38.5] - 2026-04-15

### Bug Fixes
- Reject filtered output that expands token count
([62dcc8f](https://github.com/AgusRdz/chop/commit/62dcc8f72023b97fbcfed4d6d2c65027710f28ae))
## [1.38.4] - 2026-04-13

### Features
- Add chop unwrap-hooks to reverse fix-hooks
([4c177d4](https://github.com/AgusRdz/chop/commit/4c177d4dd63818c9ee97660e7a00ef425fb034b1))
## [1.38.3] - 2026-04-13

### Bug Fixes
- Add 15 missing commands to hook interception list
([09635fd](https://github.com/AgusRdz/chop/commit/09635fd9802381a90635a9d4a1fb7656d1539362))
## [1.38.2] - 2026-04-13

### Bug Fixes
- Detect and resolve competing Bash PreToolUse hook conflicts
([b0eede7](https://github.com/AgusRdz/chop/commit/b0eede713dba28ac43c6fd7e415b95400c9571ab))
## [1.38.1] - 2026-04-06

### Bug Fixes
- Hide history legend when no 0% savings entries
([4225ec9](https://github.com/AgusRdz/chop/commit/4225ec95488f1c28ffc4d1566481e787ad82ae27))
## [1.38.0] - 2026-04-06

### Features
- Add history_compressed_only config option
([f155ff3](https://github.com/AgusRdz/chop/commit/f155ff3c4629a7e0abe1a1d148aea776054eb5c5))
## [1.37.0] - 2026-04-06

### Other
- 🛡️ Sentinel: [HIGH] Fix sensitive header leakage in curl and httpie filters
([b938b17](https://github.com/AgusRdz/chop/commit/b938b17681df7c20b485f3d8a349939122b622ca))
- 🛡️ Sentinel: [MEDIUM/HIGH] Centralized Redaction for Sensitive Data
([4232e2f](https://github.com/AgusRdz/chop/commit/4232e2fffc66c8bab0e1f4b4c83c23ddc46d756c))
- 🛡️ Sentinel: [MEDIUM/HIGH] Centralized Redaction for Sensitive Data
([a25a589](https://github.com/AgusRdz/chop/commit/a25a589c97c6183cf8a181514bea3322f8079454))
- Merge branch 'AgusRdz:main' into main
([f33df7e](https://github.com/AgusRdz/chop/commit/f33df7ec17d9dd5fba1b3f99edc931ba61030468))
- Merge branch 'AgusRdz:main' into main
([e1f91c7](https://github.com/AgusRdz/chop/commit/e1f91c73e643dad1d62ff589aa073b6c1ee080d3))
- 🛡️ Sentinel: [HIGH] Fix command injection in untrusted custom filters
([43ed5ff](https://github.com/AgusRdz/chop/commit/43ed5ff38bd7f975b8ecf081564f6a1794f6d047))
- 🧪 [testing improvement] Add tests for paths configuration
([dd47724](https://github.com/AgusRdz/chop/commit/dd47724c57a116a8c1c8e8daa616eac7639b67d2))
- Merge branch 'AgusRdz:main' into main
([aab63a9](https://github.com/AgusRdz/chop/commit/aab63a9d38bf772043a142429c2a71cd13b3d8d9))
- Merge branch 'AgusRdz:main' into main
([ee445f5](https://github.com/AgusRdz/chop/commit/ee445f53f6b441a8e41f1df9fc4d6823cba73572))
- ⚡ Optimize parseChecksum to reduce allocations
([59d469c](https://github.com/AgusRdz/chop/commit/59d469c35dbdf8f388eefa5c76145bb100d70f1e))
- ⚡ Optimize parseChecksum to reduce allocations
([2899738](https://github.com/AgusRdz/chop/commit/2899738462a2acf3803435a38a816c20c83fc119))
## [1.36.0] - 2026-04-06

### Features
- Add 14 new built-in filters for missing tools
([65c427d](https://github.com/AgusRdz/chop/commit/65c427da045ac6cb5b5d4936b9702add21d10de0))
## [1.35.4] - 2026-04-03

### Bug Fixes
- Fall through to AutoDetect when make output lacks C/gcc patterns
([4a8cd23](https://github.com/AgusRdz/chop/commit/4a8cd23a32316845352933d7f233da2195b20cee))
## [1.35.3] - 2026-04-02

### Documentation
- Update README and help text for editor config and edit commands
([c580750](https://github.com/AgusRdz/chop/commit/c58075082254a02cf0524597b68a0376ce509951))

### Miscellaneous
- Improve starter config template clarity
([cd3143f](https://github.com/AgusRdz/chop/commit/cd3143f39b48668129692462ebffd007deba1801))
## [1.35.2] - 2026-04-02

### Features
- Add chop config set to configure editor without opening a file
([544eee6](https://github.com/AgusRdz/chop/commit/544eee6c0199725e796aaa414f0a84e874365e60))
## [1.35.1] - 2026-04-02

### Features
- Add editor field to config for chop-isolated editor preference
([eea6a13](https://github.com/AgusRdz/chop/commit/eea6a13ca87a93a070850a6970b1293f11a7804c))
## [1.35.0] - 2026-04-02

### Features
- Add edit subcommand to open configs in preferred editor
([0b7e3c1](https://github.com/AgusRdz/chop/commit/0b7e3c1841574f67bb6180cfa810319ca422889e))
## [1.34.1] - 2026-04-02

### Bug Fixes
- Seed starter config with git diff disabled by default
([8b9cf81](https://github.com/AgusRdz/chop/commit/8b9cf8194d43497a1f4e9a61981b98d770119c02))

### Miscellaneous
- Name Docker volumes for go cache persistence
([9d71802](https://github.com/AgusRdz/chop/commit/9d71802412cf9448a38c8364377995e9172b5703))
## [1.34.0] - 2026-03-31

### Features
- Redesign gain UI with efficiency bar, period table, and impact bars
([2352a38](https://github.com/AgusRdz/chop/commit/2352a381866de44cba788c6dc3f824693254e366))
## [1.33.0] - 2026-03-31

### Bug Fixes
- Suppress stale update hint when cached version is not newer
([4dad622](https://github.com/AgusRdz/chop/commit/4dad62208c173e7ace0903f0a19a20d19e51c343))

### Other
- 🧪 Add tests for paths configuration
([c2338f2](https://github.com/AgusRdz/chop/commit/c2338f2be94ba79458722b1a228fb1cd0036197e))

### Performance
- Cache hot-path I/O and parallelize stats queries
([f4c8865](https://github.com/AgusRdz/chop/commit/f4c8865cd7946e3faaf0ebb7e2f54173aaebf49d))
## [1.32.1] - 2026-03-29

### Bug Fixes
- Improve network error message in updater
([a1b47cb](https://github.com/AgusRdz/chop/commit/a1b47cb46f9fe2adda0e96301b922759181eccf4))
## [1.32.0] - 2026-03-28

### Features
- Add 7 new filters and improve 3 existing ones
([915ddcf](https://github.com/AgusRdz/chop/commit/915ddcf71b36369505639916a9a2d5e52bf509e5))
## [1.31.2] - 2026-03-28

### Bug Fixes
- Harden security across updater, tracker, filters, and config
([39a85ac](https://github.com/AgusRdz/chop/commit/39a85acae19e0de783930ad57b88ab461383e217))
## [1.31.1] - 2026-03-28

### Other
- 🛡️[HIGH] Fix command injection in untrusted custom filters (#55)
([9470e0c](https://github.com/AgusRdz/chop/commit/9470e0cb4acad8a28032df94c7fd2435a5cb1ecb))
## [1.31.0] - 2026-03-27

### Bug Fixes
- Harden file permissions and optimize hot-path filters (#53)
([801efd0](https://github.com/AgusRdz/chop/commit/801efd0cb2065897fe8af3b2aaa105c5c0a2a5a7))
- Remove duplicate sensitiveHeadersRe/redactHeaders from curl.go after centralized redact merge
([10d4430](https://github.com/AgusRdz/chop/commit/10d443053f6c41aafe853bc548dd3d2c35443d9a))

### Other
- 🛡️ [MEDIUM/HIGH] Centralized Redaction for Sensitive Data (#54)
([317a91a](https://github.com/AgusRdz/chop/commit/317a91a22e6093f2029187804bbf8083b0a6b7ed))
## [1.30.3] - 2026-03-26

### Other
- 🔒 [Security Fix] Mitigate path traversal arbitrary file writes.
([c0b379c](https://github.com/AgusRdz/chop/commit/c0b379cf742ffded84d4af27a6decbc9639bc642))
- 🔒 Sentinel: [HIGH] Fix Path Traversal Vulnerability
([438b753](https://github.com/AgusRdz/chop/commit/438b753d983ea654f7585a4029f52075550c271c))
- Merge branch 'AgusRdz:main' into main
([0740947](https://github.com/AgusRdz/chop/commit/074094731254e85ff8f045a038ec7a72b6031848))
- Merge branch 'AgusRdz:main' into main
([891c1ff](https://github.com/AgusRdz/chop/commit/891c1ff469fe4476e109646d70ae0d399da2a9dd))
- 🛡️ Sentinel: [MEDIUM] Fix overly permissive directory and file permissions
([4081935](https://github.com/AgusRdz/chop/commit/4081935bfcde78a0e023d48a4e46cf017771c7d0))
- Update agent hooks to tolerate camelCase variations
([13e6524](https://github.com/AgusRdz/chop/commit/13e65244f0949247a00819f7c6d1d4802bc01506))
- Merge branch 'AgusRdz:main' into main
([c346dad](https://github.com/AgusRdz/chop/commit/c346dad1d81415a8222a2215b98ec46fb63eecae))
- Merge branch 'AgusRdz:main' into main
([5e992ac](https://github.com/AgusRdz/chop/commit/5e992ac25bdf25364695b956473d22f5e7f4b1ec))
- 🛡️ Sentinel: [HIGH] Fix sensitive header leakage in curl and httpie filters
([bee0973](https://github.com/AgusRdz/chop/commit/bee09732a64c2525506270138dd9be72a7f90dee))
- Fix sensitive header leakage in curl and httpie filters
([7e538ba](https://github.com/AgusRdz/chop/commit/7e538ba38797f28a525812a6610af4d209466146))
## [1.30.2] - 2026-03-24

### Other
- Update agents hooks for Gemini and Codex (#51)
([e157714](https://github.com/AgusRdz/chop/commit/e15771475bbea9fa28cd7dd8dd837f82eeb0d8b3))
## [1.30.1] - 2026-03-24

### Miscellaneous
- Add *.out to .gitignore
([24a43e9](https://github.com/AgusRdz/chop/commit/24a43e9bea690a3cbc12a10f950ef57aa0839a71))

### Other
- Overly permissive directory and file permissions (#50)
([548df14](https://github.com/AgusRdz/chop/commit/548df14078612416d9ca1342710f55aaba6bad35))
## [1.30.0] - 2026-03-23

### Bug Fixes
- Path traversal in runCapture filename sanitization (#48)
([3bfd246](https://github.com/AgusRdz/chop/commit/3bfd2462032f4774be234522c45b68eaf41ca89c))

### Features
- Chop global add/remove, flag-level disabled matching, colorized config output
([f86b911](https://github.com/AgusRdz/chop/commit/f86b9114b1ced1084c150be50a42f2f82d860f14))

### Other
- Remove unused function looksLikeGitPullOutput
([fdb1b1b](https://github.com/AgusRdz/chop/commit/fdb1b1b8a3fadd24e0957a26ae9fd0c5b60f288d))
- Merge branch 'AgusRdz:main' into main
([68fde93](https://github.com/AgusRdz/chop/commit/68fde93946bd45e4944ed56c426c49a3280857bc))
- Merge branch 'AgusRdz:main' into main
([869c30f](https://github.com/AgusRdz/chop/commit/869c30fad4e034a22bf333d594426c547dcfebb9))
- ⚡ Use strings.Builder for cell merging in filterAcliJiraWorkitemSearch
([e7d4df7](https://github.com/AgusRdz/chop/commit/e7d4df7bf0b79d7d8fdc1f093638c985cc0d1fca))
- 🛡️ Sentinel: [LOW] Fix Overly Permissive File Permissions
([f42ee3a](https://github.com/AgusRdz/chop/commit/f42ee3a3cfcf82eebb54865db7d6783ac4dd84b8))
- Merge branch 'AgusRdz:main' into main
([c41b896](https://github.com/AgusRdz/chop/commit/c41b89649c750e5ceac248e6f93674095c09be53))
- 🛡️ Sentinel: [HIGH] Fix path traversal in runCapture
([5da5b8f](https://github.com/AgusRdz/chop/commit/5da5b8f748f3c8b96271356eeef462de8ce57d71))
- Merge branch 'AgusRdz:main' into main
([1a1c067](https://github.com/AgusRdz/chop/commit/1a1c067f7535d3d8590d77074ddc2298615f65f1))
- 🔒 [Security Fix] Mitigate path traversal arbitrary file writes.
([ba9e95c](https://github.com/AgusRdz/chop/commit/ba9e95ce626d987417b0babc830b1fbf8247c63d))
- 🔒 Sentinel: [HIGH] Fix Path Traversal Vulnerability
([d696011](https://github.com/AgusRdz/chop/commit/d696011722f532ac700b7d0ea48fbed85dfe198b))

### Testing
- Add comprehensive edge case tests for splitLogical
([61dee41](https://github.com/AgusRdz/chop/commit/61dee41d869b89d987efde171dd7cf3b2a6a6536))
## [1.29.0] - 2026-03-23

### Bug Fixes
- Harden directory permissions from 0755 to 0700
([8bb6025](https://github.com/AgusRdz/chop/commit/8bb6025dd477a062e69c3553a7dcb996c3d2cb5e))

### Testing
- Add comprehensive coverage tests across all packages
([716bd3d](https://github.com/AgusRdz/chop/commit/716bd3dd8b910ae9e987473db5d0ef720027e0a2))
## [1.28.1] - 2026-03-23

### Documentation
- Document completion, config export/import, filter new, and per-project stats
([680c9f4](https://github.com/AgusRdz/chop/commit/680c9f41712f41a2abdea3a61bcd0b28b4db0f0b))
- Reorganize README and extract detailed reference to docs/
([993d873](https://github.com/AgusRdz/chop/commit/993d8739ef12fb0149b761b69a35ff6bba613d8e))
- Fix shell completion install paths for macOS vs Linux
([9a579e9](https://github.com/AgusRdz/chop/commit/9a579e96b9ada0c1d1cb97ca3b99f2a185aa1e04))

### Miscellaneous
- Add .gitattributes to normalize line endings to LF
([4dc1a01](https://github.com/AgusRdz/chop/commit/4dc1a01678f0105743289657cc0b06c457391762))
- Remove .gitattributes
([98c3f80](https://github.com/AgusRdz/chop/commit/98c3f80efa0f0718d7ef60919b7bf3403d885dda))

### Other
- Remove unused function looksLikeGitPullOutput (#44)
([3c6f035](https://github.com/AgusRdz/chop/commit/3c6f035c4b5ffb0dec54dc0def09334b0624f90a))
- ⚡ Use strings.Builder for cell merging in filterAcliJiraWorkitemSearch (#45)
([c9c0240](https://github.com/AgusRdz/chop/commit/c9c02407e92cfb95c87072120192232243e19c05))
- 🛡️ Sentinel: [LOW] Fix Overly Permissive File Permissions (#46)
([3944410](https://github.com/AgusRdz/chop/commit/39444103dbe4589e2f4607d9e2a515093795fb78))
- Fix:path traversal in runCapture (#47)
([676b464](https://github.com/AgusRdz/chop/commit/676b464fa76984fa7ce3cded7e2d49244269cadd))

### Testing
- Add comprehensive edge case tests for splitLogical (#43)
([619b637](https://github.com/AgusRdz/chop/commit/619b63716e760fe70b85686f3d46389d43d43252))
## [1.28.0] - 2026-03-19

### Features
- Shell completions, per-project stats, config sync, security fixes, and help redesign
([4c38a91](https://github.com/AgusRdz/chop/commit/4c38a91a74eb833338887ba7d44d45b1ea6c13f6))
## [1.27.0] - 2026-03-19

### Bug Fixes
- Never expand output beyond raw size
([740d84a](https://github.com/AgusRdz/chop/commit/740d84a3e54b027185ddd4c344434809c79efb55))

### Documentation
- Bust logo cache
([a306c12](https://github.com/AgusRdz/chop/commit/a306c12db7f117a0bab35469ceaa476cc9cc330b))
- Remove obsolete CLAUDE.md snippet from Claude Code section
([732c86b](https://github.com/AgusRdz/chop/commit/732c86bd104e07e37e285de8c54600f35f5f1333))
## [1.26.0] - 2026-03-18

### Documentation
- Document agent-info, agent-handshake, and setup alias
([a7a1a91](https://github.com/AgusRdz/chop/commit/a7a1a91e37979e22f9b645fde4af9e7c74ba24ea))

### Features
- Make chop AI-native with agent discovery and introspectability
([64cb009](https://github.com/AgusRdz/chop/commit/64cb00906cc20ef2c85024b6378efe68438e9dd6))
- Agent-first discovery with handshake and security fix (#42)
([c6b7bfc](https://github.com/AgusRdz/chop/commit/c6b7bfc0edad9888a06cfbff0ec5eca314e05221))

### Miscellaneous
- Ignore chop binary in git
([a390ac9](https://github.com/AgusRdz/chop/commit/a390ac96d0c95526da0ca36aeb4c79ee76e716e8))
- Update logo
([cd772d8](https://github.com/AgusRdz/chop/commit/cd772d8325539d93f0572d5ce309e084b2cb856e))
## [1.25.2] - 2026-03-17

### Bug Fixes
- Improve PATH handling on Windows for immediate availability
([023e455](https://github.com/AgusRdz/chop/commit/023e45509daaff96cd817d97840578fed381e193))

### Documentation
- Update README with Antigravity IDE support, systemctl filter, and contributors
([d840da1](https://github.com/AgusRdz/chop/commit/d840da1e473bc79e813456c61b5ba48fbc412d67))

### Other
- Merge pull request #38 from giankpetrov/main
([869cb9a](https://github.com/AgusRdz/chop/commit/869cb9a17a0679a3dabc1e1046d4bc673cc0867a))
## [1.25.1] - 2026-03-17

### Bug Fixes
- Migrate Windows tracking DB from legacy ~/.local/share/chop path
([bc24f03](https://github.com/AgusRdz/chop/commit/bc24f036ebe53a9d2cadb391a3a767efb5a0be9c))
## [1.25.0] - 2026-03-17

### Bug Fixes
- Restrict world-writable bypass in IsSecure to /tmp and /var/tmp
([3c585da](https://github.com/AgusRdz/chop/commit/3c585da57b724cba47f5daa88f60442a9386ef4f))

### Features
- Add support for Antigravity IDE integration
([9c6ffa2](https://github.com/AgusRdz/chop/commit/9c6ffa236f84272897dfea8677bf9a31fff8c909))
- Add systemctl filter
([426d094](https://github.com/AgusRdz/chop/commit/426d0943cb1b8ee709613925c3eabc8d7ef69693))
- Add systemctl filter
([82f4c46](https://github.com/AgusRdz/chop/commit/82f4c4697e8da22cec951115ba60c78e19a24bcd))

### Miscellaneous
- Remove .jules AI agent artifacts and gitignore .jules/
([c4b2e77](https://github.com/AgusRdz/chop/commit/c4b2e77a561aa7c4468371253a129bd0ebf20360))

### Other
- Fix world-writable path bypass in /tmp
([3577577](https://github.com/AgusRdz/chop/commit/3577577292d85aa1886733dca2e5b6b9b2101414))
- Merge pull request #36 from giankpetrov/main
([2fba55f](https://github.com/AgusRdz/chop/commit/2fba55f523341ecbac02959ae3a3df4e1372c6e9))
- Merge pull request #37 from giankpetrov/feat/systemctl-filter-4588287500168083270
([5be40bf](https://github.com/AgusRdz/chop/commit/5be40bfea0e1b2cb27b2b86cce8dd0149a7ac83a))
## [1.24.0] - 2026-03-16

### Other
- Harden release signing and add public key
([d4eb28e](https://github.com/AgusRdz/chop/commit/d4eb28e3f2c2abe811f33d9f6a25e43544d08e40))
## [1.23.0] - 2026-03-16

### Other
- Add Ed25519 signature verification for updates
([1622cd5](https://github.com/AgusRdz/chop/commit/1622cd5d2a7e5045115a13d82936caca1f6046c6))
## [1.22.2] - 2026-03-16

### Other
- Refactor git_status.go to remove unused sectionNone constant
([e275f37](https://github.com/AgusRdz/chop/commit/e275f3795dcf647e5c2c5a7954c7b42eb3d93d26))
- Merge branch 'AgusRdz:main' into main
([5e9e444](https://github.com/AgusRdz/chop/commit/5e9e4448af9cf9af18db56322093c58e30ac523e))
- Merge pull request #33 from giankpetrov/main
([2b8757d](https://github.com/AgusRdz/chop/commit/2b8757de92ce240ea35ea3a74923b9713b1c8db8))
## [1.22.1] - 2026-03-16

### Other
- Merge pull request #32 from giankpetrov/main
([f551e21](https://github.com/AgusRdz/chop/commit/f551e213f4cd23dbe0480bd121c1e1eaa5a5a7f4))

### Performance
- Optimize npm test filter by pre-compiling regexes
([c5b67a2](https://github.com/AgusRdz/chop/commit/c5b67a29128fbf50eed91ae5a3a10b5f1ab0dd9c))
## [1.22.0] - 2026-03-16

### Features
- Feat: add support for Gemini CLI and Codex CLI
([549e9d1](https://github.com/AgusRdz/chop/commit/549e9d1c0d89c5613fc96d390f87cb80fc71715c))

### Other
- Secure SQL LIKE queries against injection
([b5b0477](https://github.com/AgusRdz/chop/commit/b5b0477d00b2d3568ee015d34569b8f818b27570))
- Secure SQL LIKE queries against injection (#30)
([efa1ff2](https://github.com/AgusRdz/chop/commit/efa1ff2966ce32cea313d3c4a531a93222e34461))
- Merge pull request #31 from giankpetrov/main
([ed7592d](https://github.com/AgusRdz/chop/commit/ed7592d9f25f2b4835c02bb208000018b6d3e641))
## [1.21.0] - 2026-03-16

### Bug Fixes
- Correct stale branch comparison logic (#28)
([89590f8](https://github.com/AgusRdz/chop/commit/89590f8171996f4370116f3ceb9a7f71d21e7de7))

### Features
- Add native windows support and platform-aware paths
([36fb7a4](https://github.com/AgusRdz/chop/commit/36fb7a474fc916d93f19e482c05fce9b6b6af255))
- Add native windows support and platform-aware paths (#29)
([d8c1302](https://github.com/AgusRdz/chop/commit/d8c13025168dbb469fe87e152c27376451775b03))
## [1.20.0] - 2026-03-16

### Bug Fixes
- Add issues:write permission for PR comment posting
([ea2439a](https://github.com/AgusRdz/chop/commit/ea2439a1d699e8b622e2261122d760adf7afdc54))
- Use pull_request_target to allow comment posting from fork PRs
([150ea79](https://github.com/AgusRdz/chop/commit/150ea795b52804105a0d72024f23744a72e744f3))
- Gracefully handle missing permissions in stale branch check
([3e9990c](https://github.com/AgusRdz/chop/commit/3e9990c5a73d472629990382de5ecb086ae8a9e2))

### Features
- Add Gemini CLI support (#26)
([dd505af](https://github.com/AgusRdz/chop/commit/dd505af0c54ab72635b4c3fc208655e151cdf5bc))

### Other
- Fix potential SQL wildcard injection in tracking queries
([d43393c](https://github.com/AgusRdz/chop/commit/d43393c7165ce9f3d5bf7a769c34b6e7e7786922))
## [1.19.0] - 2026-03-15

### CI/CD
- Add stale branch check workflow for PRs
([d8666f6](https://github.com/AgusRdz/chop/commit/d8666f6b074d53de8b45dfdc295baf94b7617510))
- Opt into Node.js 24 for stale-check workflow
([40584da](https://github.com/AgusRdz/chop/commit/40584da9aa7204d17969cc6c426feae77227b4dc))
- Fix FORCE_JAVASCRIPT_ACTIONS_TO_NODE24 to use string value
([96826fb](https://github.com/AgusRdz/chop/commit/96826fbf68a3ccbb6b2b1ee26dc24ceb967167bb))
- Update actions/checkout to v5 and actions/setup-go to v6 for Node.js 24 support
([aacaf79](https://github.com/AgusRdz/chop/commit/aacaf7964b3551cf85db4df2d7b78cd91404ab2d))

### Documentation
- Add PR template and fork sync guide to CONTRIBUTING.md
([68c2c17](https://github.com/AgusRdz/chop/commit/68c2c1798483ac58c9a03bac8e155e7123f243ba))
- Clarify fork sync steps for both upstream remote and GitHub UI workflows
([7613e90](https://github.com/AgusRdz/chop/commit/7613e90f6285e9262d921d07b65be8d6db569e9a))

### Features
- Add chop diff command for raw vs filtered comparison (#25)
([644dd75](https://github.com/AgusRdz/chop/commit/644dd750f4d957c1f64efaaddc1df2bb841c6d80))

### Refactoring
- Replace filter switch with registry pattern (#24)
([3ffcda2](https://github.com/AgusRdz/chop/commit/3ffcda201251626f09e45b183feda26ac306d6b4))
## [1.18.5] - 2026-03-15

### Documentation
- Add CONTRIBUTING.md (#23)
([d569bac](https://github.com/AgusRdz/chop/commit/d569bac230a68e0e6a3a3c8b18be5aab7cb56a63))

### Features
- Add podman and opentofu aliases (#22)
([845e82a](https://github.com/AgusRdz/chop/commit/845e82a094cce3fba388ad7b4c281d5fb6e7b262))
## [1.18.4] - 2026-03-15

### Bug Fixes
- Secure file permissions and optimize user filter string building (#19, #20)
([9743e9a](https://github.com/AgusRdz/chop/commit/9743e9a67a646a649ead2ac989f06706a41779a9))

### Features
- Add git pull/fetch filters and chop list command (#21)
([f9cdf8f](https://github.com/AgusRdz/chop/commit/f9cdf8ff1e00c2486c755d057088e2e00c714955))

### Testing
- Expand custom filter config tests with trust model and edge cases (#17)
([8431083](https://github.com/AgusRdz/chop/commit/8431083df9453859696e155c2f94f155e035f1f6))
- Add tests for updater auto-update configuration (#18)
([8354a21](https://github.com/AgusRdz/chop/commit/8354a2134d33ed0edd5b67e54e02dc5d4a3a900c))
## [1.18.3] - 2026-03-15

### Bug Fixes
- Replace sh -c with direct exec to prevent command injection in custom filters (#15)
([d6d417b](https://github.com/AgusRdz/chop/commit/d6d417bb291a795264e5d181b563cdfeb71861a1))
- Proper large number formatting and SHA256 checksum verification
([fe72971](https://github.com/AgusRdz/chop/commit/fe72971418cb1c0417421932d64e57176bd51374))

### Performance
- Optimize string concatenation in kubectl logs filter (#16)
([474aa96](https://github.com/AgusRdz/chop/commit/474aa96f310349638f468c8b8aa1de17aeaa68f9))
## [1.18.2] - 2026-03-15

### Miscellaneous
- Remove unused loop index in npm_test_cmd.go (#11)
([1660c22](https://github.com/AgusRdz/chop/commit/1660c22cb6d86786bd0dadbdfb923ede4ca58e03))

### Performance
- Optimize Playwright filter by pre-compiling regular expressions (#12)
([1ccd74e](https://github.com/AgusRdz/chop/commit/1ccd74edefe5b90dfc9b892a4787a143befa8f5b))

### Testing
- Add tests for LoadCustomFilters and FiltersConfigPath (#13)
([2d4f84c](https://github.com/AgusRdz/chop/commit/2d4f84c757660bca9aa57cdf8b63aa6157d9dc85))
## [1.18.1] - 2026-03-14

### Bug Fixes
- Preserve raw output for failed commands
([c761005](https://github.com/AgusRdz/chop/commit/c761005c622a5f654d041f8d56070159aad4c64d))
## [1.18.0] - 2026-03-13

### Features
- Add ansible-playbook, docker-compose, and kubectl JSON filters (#10)
([c1df053](https://github.com/AgusRdz/chop/commit/c1df053ba26cc5588fc312fb00e1f8ac4d46cc4f))
## [1.17.0] - 2026-03-13

### Testing
- Enhance test coverage and add coverage command
([7b58a76](https://github.com/AgusRdz/chop/commit/7b58a76e09f25c23c0592685091e6d812cb5839f))
## [1.16.0] - 2026-03-13

### Features
- Show per-period compression % in gain report
([353da52](https://github.com/AgusRdz/chop/commit/353da52d459f44889dd8eb157b242efde1cfd746))
## [1.15.0] - 2026-03-13

### Features
- Truncate long commands in gain --history, add --verbose for full output
([09939d5](https://github.com/AgusRdz/chop/commit/09939d524989e259c5af3c02ba17ed89f5264efd))
## [1.14.0] - 2026-03-13

### Features
- Add --limit and --all flags to gain --history, support combining with --since
([47a7260](https://github.com/AgusRdz/chop/commit/47a7260b36dfeef3347ecd19c87ae180b2461283))
## [1.13.0] - 2026-03-13

### Features
- Add `chop auto-update on/off` toggle with update notifications (#8)
([a2adab7](https://github.com/AgusRdz/chop/commit/a2adab714a455c7700175e949e8bdb239218600e))
## [1.12.0] - 2026-03-13

### Features
- Add chop gain --export json|csv and --since <duration>
([bf357f2](https://github.com/AgusRdz/chop/commit/bf357f24bb0636e0a7c04b31da656581bdcbe727))
## [1.11.1] - 2026-03-13

### Bug Fixes
- Wrap commands with fd redirects like 2>&1
([b32d5ba](https://github.com/AgusRdz/chop/commit/b32d5baf721a6a73478e76266c7c26bcb7779679))
## [1.11.0] - 2026-03-13

### Features
- Add artifact attestations and homebrew tap
([d173cf2](https://github.com/AgusRdz/chop/commit/d173cf249bf3c1d287281aee257a0b42009571dc))
## [1.10.4] - 2026-03-13

### Features
- Add `chop enable` and `chop disable` global kill switch
([908a87a](https://github.com/AgusRdz/chop/commit/908a87a00838463a6839b791c221902240cc4b02))

### Other
- Merge pull request #7 from aeanez/feat/enable-disable
([b859406](https://github.com/AgusRdz/chop/commit/b8594061c0d687549a2700d40be70cf8f3fe7608))
## [1.10.3] - 2026-03-13

### Bug Fixes
- Handle quoted paths in shouldWrap
([49677ca](https://github.com/AgusRdz/chop/commit/49677ca8bd5f37229ab66cadf67e69a4a95819d8))

### Other
- Merge pull request #1 from giankpetrov/improve-hook-parsing-16568673576002575404
([b0d8c84](https://github.com/AgusRdz/chop/commit/b0d8c84cb4df26f2f6e7aa9a3366c1287bf39e72))
- Merge pull request #6 from giankpetrov/main
([ffa0384](https://github.com/AgusRdz/chop/commit/ffa0384da15167c155bb1c384f94a5b0e6caa901))
## [1.10.2] - 2026-03-13

### Bug Fixes
- Replace \n with colon in mermaid node labels
([7036109](https://github.com/AgusRdz/chop/commit/70361097c2afdaee2baa137c36eee5b0cf6744f7))

### Documentation
- Add How It Works, Cascade Effect, and token cost sections to README
([c4b45d8](https://github.com/AgusRdz/chop/commit/c4b45d8c5a6a9ad2cc3446854e1be3271241b9b7))
- Explain ! marker in history and how to remove noisy write commands
([f81085c](https://github.com/AgusRdz/chop/commit/f81085cc79493cb3311b239144f8f1348338be5b))

### Features
- Add --no-track and --resume-track to permanently suppress command tracking
([c6c0927](https://github.com/AgusRdz/chop/commit/c6c0927f3f093bc1ee7a4b7ee938258e0808663b))
## [1.10.1] - 2026-03-13

### Bug Fixes
- Prevent tracking data loss under concurrent chop processes
([30d70ed](https://github.com/AgusRdz/chop/commit/30d70ed6724b0c51934214bea023bdbf79c10bf6))
## [1.10.0] - 2026-03-12

### Features
- Compress git diff --stat output, fix grep filter expansion
([ae91b8e](https://github.com/AgusRdz/chop/commit/ae91b8efe816994cc4d608c9b772a080c3a96765))
## [1.9.0] - 2026-03-12

### Features
- Quote-aware operator parsing in hook
([7755e75](https://github.com/AgusRdz/chop/commit/7755e7518862fda0e0565946a429a59f8bfbfc7c))
## [1.8.0] - 2026-03-12

### Bug Fixes
- Regenerate CHANGELOG.md before build, skip housekeeping commits
([214c645](https://github.com/AgusRdz/chop/commit/214c6453afe738f40277b638c7c3e4ef0fdfcd15))

### Features
- Wrap compound commands in hook, split on logical operators
([fe131ba](https://github.com/AgusRdz/chop/commit/fe131ba0f2744142cdef805c9ed5a8f2b55a1ce1))
## [1.7.0] - 2026-03-12

### CI/CD
- Use CHANGELOG.md as GitHub Release body
([dffd0c8](https://github.com/AgusRdz/chop/commit/dffd0c82a6b698d34c209a4f46e0cfffed77b496))

### Documentation
- Add CHANGELOG.md covering v1.0.0 through v1.6.0
([d9993a1](https://github.com/AgusRdz/chop/commit/d9993a1a0548c9e72874361497320c308e572a9f))

### Features
- Add changelog generation and `chop changelog` command
([68c6a8a](https://github.com/AgusRdz/chop/commit/68c6a8ac7a785ef38a0792b7ca6a06fee8769614))
- Add auto-update - check and apply updates automatically
([c6edf57](https://github.com/AgusRdz/chop/commit/c6edf575102bfe9a037367785febd2353d9f20b1))
- Invert changelog defaults, add tests for cleanup and updater
([3baaaf8](https://github.com/AgusRdz/chop/commit/3baaaf88b899cec880efff1c3597fac7872f90e1))

### Miscellaneous
- Ignore docs dir, opt into Node.js 24 for GitHub Actions
([e4015eb](https://github.com/AgusRdz/chop/commit/e4015ebb3bc5cf5da9fdaec159dbb1b4d3f6f683))
## [1.6.0] - 2026-03-12

### Documentation
- Update README for v1.5.0 — new filters, unchopped report flags
([36c4cf3](https://github.com/AgusRdz/chop/commit/36c4cf33139a565e4c8d5d3588bbc6405cfba23d))

### Features
- Add user-defined custom filters via filters.yml
([17fdd7f](https://github.com/AgusRdz/chop/commit/17fdd7f6eb6d94827ee17b789dd38032858a0117))
- Add user-defined custom filters via filters.yml (#3)
([4ffdcec](https://github.com/AgusRdz/chop/commit/4ffdcec19af01d9b1b5ba9839688edbf38afef7f))
- User-defined custom filters, config init, em-dash cleanup
([ee37a29](https://github.com/AgusRdz/chop/commit/ee37a2973ecf79670ef7142c3060ab49631d8727))
## [1.5.0] - 2026-03-12

### Features
- Expand filter coverage and improve unchopped report UX
([1f86f33](https://github.com/AgusRdz/chop/commit/1f86f33ce02028338eab33803a996e117c2cef6b))
## [1.4.0] - 2026-03-11

### Documentation
- Document log pattern compression and cat/tail/less/more support
([29e2543](https://github.com/AgusRdz/chop/commit/29e2543ec76c1a150cd34a3c8972ee6484a50fee))

### Features
- Add `chop gain --unchopped` to identify commands without filters
([fb6dd46](https://github.com/AgusRdz/chop/commit/fb6dd4659b1c188d46d0e33a43b653c539b4451c))
- Add chop gain --unchopped to identify commands without filters
([29babe7](https://github.com/AgusRdz/chop/commit/29babe71b51ecc358e28f0b0628e1b0208daea5a))
## [1.3.0] - 2026-03-11

### Bug Fixes
- Show real example line instead of fingerprint for repeated log patterns
([14163a4](https://github.com/AgusRdz/chop/commit/14163a4a704e379e8a115657387b667b02e6e072))

### Features
- Add pattern-based log compaction for repetitive log lines
([c77a78a](https://github.com/AgusRdz/chop/commit/c77a78ade53d6c67386a42b72c1553eb8f29e643))
- Add pattern-based log compaction for repetitive log lines (#1)
([6ed5d00](https://github.com/AgusRdz/chop/commit/6ed5d0068f4a27bac428350527e72c4c1fab3725))
## [1.2.2] - 2026-03-10

### Bug Fixes
- Panic on docker ps with custom --format table
([8303380](https://github.com/AgusRdz/chop/commit/830338067f2425ad9c7c8f096723d2d8250cfda7))

### Documentation
- Add npm test before/after example, fix token count
([8e637dc](https://github.com/AgusRdz/chop/commit/8e637dc14025ae26975514ce3d55ea4c33f67bd5))
## [1.2.1] - 2026-03-10

### Bug Fixes
- Use local timezone instead of UTC for gain stats
([64bf9da](https://github.com/AgusRdz/chop/commit/64bf9da5d455d384f417dfb0738b793c07b587c2))

### Miscellaneous
- Add run-name to release workflow
([0588bc2](https://github.com/AgusRdz/chop/commit/0588bc22fa87d42574114457618444a8b0145324))
## [1.2.0] - 2026-03-09

### Features
- Add npx playwright/tsc/ng, acli jira, node, and find filter support
([cc97ffb](https://github.com/AgusRdz/chop/commit/cc97ffbe53342c9434b48b5c12c71e0ed2599805))
## [1.1.0] - 2026-03-09

### Features
- Subcommand-level disabled config, local .chop.yml, section-aware git status
([47a6d29](https://github.com/AgusRdz/chop/commit/47a6d29c97cab7d56bbb35599513ab4050716fc4))
## [1.0.5] - 2026-03-09

### Bug Fixes
- Use calendar-based periods for gain stats (week=Mon-Sun, month=1st, year=Jan1)
([8d1e732](https://github.com/AgusRdz/chop/commit/8d1e732b8dc32bbaa2d22399c23ba46838fea638))
## [1.0.4] - 2026-03-09

### Features
- Add chop doctor command to detect and fix hook path mismatches
([e724941](https://github.com/AgusRdz/chop/commit/e72494150c2f97ae1cad1b4d30e22b5c136d0cc2))
## [1.0.3] - 2026-03-09

### Features
- Add weekly/monthly/yearly metrics to chop gain, fix Windows install path
([ebcb2ba](https://github.com/AgusRdz/chop/commit/ebcb2baf22ba64d12bb052185a0b96c06dff1909))
## [1.0.2] - 2026-03-09

### Bug Fixes
- Auto-add to PATH on Windows during install
([aefe521](https://github.com/AgusRdz/chop/commit/aefe5216df9d704cf3420c3c5d0570f8c273c06c))
## [1.0.1] - 2026-03-08

### Bug Fixes
- Handle git global flags (-C, --no-pager, etc.) before subcommand matching
([86555da](https://github.com/AgusRdz/chop/commit/86555da4b1970df4cdf43daad4a4ab725833bee8))

### Documentation
- Update migration note to reference pre-v1.0.0
([15fd2e7](https://github.com/AgusRdz/chop/commit/15fd2e77b30e1a7b719da94934341fce5efe8bc0))
## [1.0.0] - 2026-03-08

### Features
- Add Windows native support — install.ps1, migrate.ps1, PATH management
([1e358b3](https://github.com/AgusRdz/chop/commit/1e358b39dc3d41473bcf7fa070a68923732b5e85))
## [0.14.7] - 2026-03-08

### Documentation
- Document --post-update-check flag in help and README
([b34632c](https://github.com/AgusRdz/chop/commit/b34632cc852578887f3fc7cd649c9c18eb1ffca7))
## [0.14.6] - 2026-03-08

### Bug Fixes
- Correct env var syntax for versioned and custom-dir installs
([0bad163](https://github.com/AgusRdz/chop/commit/0bad163f3865f2ac6c6f099444841d9672fe2896))
- Re-exec new binary after update to show migration hint regardless of source version
([cff1be3](https://github.com/AgusRdz/chop/commit/cff1be32dc36be1580fad1e78f77743e72cdf165))
## [0.14.5] - 2026-03-07

### Features
- Suggest migration after update when installed in legacy ~/bin
([b5b64fc](https://github.com/AgusRdz/chop/commit/b5b64fceb82ac699392653921f44164515d5d4a4))
## [0.14.4] - 2026-03-07

### Bug Fixes
- Change default install dir to ~/.local/bin, add migration script
([54bdee2](https://github.com/AgusRdz/chop/commit/54bdee20383da6aa64dea92c45c8f17ac9dcd75b))
## [0.14.3] - 2026-03-07

### Bug Fixes
- Auto-add install dir to shell config when not in PATH
([fc1cf98](https://github.com/AgusRdz/chop/commit/fc1cf98bc60a682d3380f66e58d54f2752c801fc))
## [0.14.2] - 2026-03-07

### Bug Fixes
- Show persistent PATH setup instructions after install
([59d8510](https://github.com/AgusRdz/chop/commit/59d85109c831716a717788b7330e913eda80833a))

### Documentation
- Add name explanation and document uninstall/reset commands
([50269ee](https://github.com/AgusRdz/chop/commit/50269ee46e20eb8e157ec23ca82548035c3782ee))
## [0.14.1] - 2026-03-06

### Documentation
- Improve disabled config documentation in README and help
([191524b](https://github.com/AgusRdz/chop/commit/191524bd848a38560e40c691543fff33bf5ed2fc))

### Features
- Add filter routing for git show, stash list, ng/nx lint, dotnet clean/pack/publish
([eb08e50](https://github.com/AgusRdz/chop/commit/eb08e509345b145d11df761217886d8799591509))
## [0.14.0] - 2026-03-06

### Refactoring
- Remove dead code — read, shell, discover, tee packages
([0169ebf](https://github.com/AgusRdz/chop/commit/0169ebf8b66430871076702c3a123c6455e4150d))
## [0.13.0] - 2026-03-06

### Features
- Add chop uninstall and chop reset commands
([79926c8](https://github.com/AgusRdz/chop/commit/79926c83d1647de23db00380279f21451fe356cf))
## [0.12.0] - 2026-03-06

### Features
- Claude-only focus, add init --status, remove shell integration
([21c698f](https://github.com/AgusRdz/chop/commit/21c698f44e9848ad58df4c9c3d346a7815a59f6f))
## [0.11.0] - 2026-03-06

### Features
- Stdin support for chop read, comprehensive README rewrite
([7773332](https://github.com/AgusRdz/chop/commit/77733327f01d8fa8225e9a0d676a420bd58bdc82))
## [0.10.1] - 2026-03-06

### Testing
- Enrich test fixtures with realistic output for 8 filters
([6e37c2b](https://github.com/AgusRdz/chop/commit/6e37c2bdf5816fd7ca689c885fa9559248414df0))
## [0.10.0] - 2026-03-06

### Features
- Add chop update command for self-updating
([c758b80](https://github.com/AgusRdz/chop/commit/c758b809ebf0506b90c287b95e36d998f4fda000))
## [0.9.0] - 2026-03-06

### Documentation
- Update README for any-agent usage, add post-install instructions
([cf44606](https://github.com/AgusRdz/chop/commit/cf44606277d18038d56e8749a86a02a855c4b36e))

### Features
- Add PowerShell shell integration
([292ad75](https://github.com/AgusRdz/chop/commit/292ad759cd7bebbb845d556117096299e595ae42))
## [0.8.0] - 2026-03-06

### Bug Fixes
- Avoid stripping /* */ inside string literals in chop read
([4c5a0b6](https://github.com/AgusRdz/chop/commit/4c5a0b6f4e8bee38424d6f58e50730f01fe49951))

### Features
- Add install.sh for one-line binary installation
([eef3156](https://github.com/AgusRdz/chop/commit/eef3156efa39180adc7a308b2b1a611eedb1b491))

### Miscellaneous
- Switch to GitHub origin with CI and release workflows
([aeefd26](https://github.com/AgusRdz/chop/commit/aeefd26fb762b59c01a931b36e028d64f8e5c04a))
## [0.7.0] - 2026-03-06

### Features
- Chop read — language-aware file compression
([72690b5](https://github.com/AgusRdz/chop/commit/72690b5658df8ddd5681be44c61f43d4d3aad72a))
## [0.6.0] - 2026-03-06

### Features
- Hook audit logging and discover command
([d8488d9](https://github.com/AgusRdz/chop/commit/d8488d95db855ef3b4df52efe27b52c5da400587))
## [0.5.1] - 2026-03-05

### Features
- Per-command summary and 0% highlighting in gain metrics
([c18b27d](https://github.com/AgusRdz/chop/commit/c18b27d98360cd38090fc68aef3a595ec560a410))
## [0.5.0] - 2026-03-05

### Features
- Claude Code hook integration
([31542a5](https://github.com/AgusRdz/chop/commit/31542a537882f3f949956e165ce70f500b56af07))
## [0.4.2] - 2026-03-05

### Bug Fixes
- Docker images new format, git branch summarization
([a34cbcb](https://github.com/AgusRdz/chop/commit/a34cbcbc81faeb7718beb171649453d9ee78836d))
## [0.4.1] - 2026-03-05

### Features
- Add help command
([079ef23](https://github.com/AgusRdz/chop/commit/079ef233c8e8ebff7903d36af59e1306b59ba223))
## [0.4.0] - 2026-03-05

### Bug Fixes
- Auto-detect host OS in Makefile install target
([69dfab3](https://github.com/AgusRdz/chop/commit/69dfab36f29eaf5223b42e48f990e9491bf4c693))

### Features
- Semver release targets and CI tag validation
([6133708](https://github.com/AgusRdz/chop/commit/61337081979cbcccc3c88fb8d683fbc0c50d82ff))
- Config file support and shell integration
([d35bba9](https://github.com/AgusRdz/chop/commit/d35bba9070f6f1af6bcb4b3d2c6254ba36fb87ca))
## [0.3.0] - 2026-03-05

### Features
- Add 40+ filters, auto-version from tags, simplified CI
([d9e3387](https://github.com/AgusRdz/chop/commit/d9e338725be0549d1610f797d424f8230eb628f3))
## [0.2.0] - 2026-03-05

### Features
- Tee mode, capture mode, sanity guards on all 52 filters
([9e1847c](https://github.com/AgusRdz/chop/commit/9e1847cb078fbda547455dc7e97c9126c70851cc))
## [0.1.0] - 2026-03-05

### Documentation
- Add README, LICENSE, CI pipeline, Makefile
([bec9741](https://github.com/AgusRdz/chop/commit/bec97416f3230f1e3c4dbaf6259830e4f0038a4e))

### Features
- Initial chop CLI with 25 filters and token tracking
([0382705](https://github.com/AgusRdz/chop/commit/0382705390110df3a72a873b9579df0ae1568e3e))
- Add gh CLI, grep/rg filters — 37 total filters, 129 tests
([c5e67e1](https://github.com/AgusRdz/chop/commit/c5e67e1d4e725d3db60eb3d84b99c11d0e005f71))
- Add auto-detect, cloud CLIs, java build tools
([4272d1d](https://github.com/AgusRdz/chop/commit/4272d1d5704da24924b310cd7b165ba0a0499746))

