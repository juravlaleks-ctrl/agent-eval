# Contributing to `agent-eval`

`agent-eval` keeps a strict split between the public English repository surface and the Russian engineering/business documentation set.

This guide reflects the current scope: MCP-native deterministic local evaluation for release gating.

## Scope Discipline

- Keep proposals within the MCP-native local gating wedge unless a roadmap milestone explicitly allows expansion.
- Avoid broad "evaluation platform" feature pushes unless they improve release decision fidelity.
- Ensure implementation claims in docs match source reality.

## Documentation Rule

- Public files allowed on `main` must remain in English:
  - `README.md`
  - `LICENSE`
  - `NOTICE`
  - `CONTRIBUTING.md`
- Engineering and business documentation must remain in Russian under `docs/`.
- The Russian documentation set must not be merged into `main`.
- If behavior changes, update the Russian documentation stack together on the active development branch:
  - `docs/SPEC.md`
  - `docs/VISION.md`
  - `docs/BUSINESS-REQUIREMENTS.md`
  - `docs/ARCHITECTURE.md`
  - `docs/ROADMAP.md`
  - `docs/GOVERNANCE.md`
  - `docs/EXTENSIBILITY.md`
  - `docs/IMPLEMENTATION-STATUS.md`

`README.md` must remain accurate regarding the current implementation status and executable behavior.

## Contribution Process

1. Describe intent against the current phase in the roadmap.
2. Align with stability rules in `docs/GOVERNANCE.md` on the active development branch.
3. Add/update requirement traces in `docs/BUSINESS-REQUIREMENTS.md` and `docs/ARCHITECTURE.md`.
4. Add implementation notes only when file/module reality changes.
5. Run focused verification for touched code/tests before proposing merge.

## License and Attribution

- All contributions are subject to [Apache-2.0](./LICENSE).
- Do not remove or alter license/notice obligations in `LICENSE` and `NOTICE`.
- If you add dependencies, include compatible attribution in `NOTICE` and preserve license files in redistributions.
