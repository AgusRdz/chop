# Changelog

All notable changes to chop are documented here.

## [Unreleased]

### Features
- Add changelog generation and `chop changelog` command
([68c6a8a](https://github.com/AgusRdz/chop/commit/68c6a8ac7a785ef38a0792b7ca6a06fee8769614))
## [1.6.0] - 2026-03-12

### Bug Fixes
- Address PR #3 review findings in custom filters
([e09e409](https://github.com/AgusRdz/chop/commit/e09e409c4999c5aea67a481b7757442b6ff968ab))

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

