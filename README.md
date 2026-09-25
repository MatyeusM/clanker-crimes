# Clanker Crimes

A static spot-the-bad-AI-code game. Each case is a working portfolio project with
subtle code smells hiding in the source. Browse the files, select every crime you
find, and submit your verdict to advance.

Play the deployed version at [matyeusm.github.io/clanker-crimes](https://matyeusm.github.io/clanker-crimes/).

## How it works

- Cases are generated from the projects in `packages/` and published as static
  case data under `dist/case/`.
- The case viewer shows a file tree, theme-aware syntax-highlighted source, and
  a shuffled list of possible crimes.
- A verdict is converted into a SHA-256 digest in the browser. The matching
  generated file either advances to the next case or shows the final results
  screen; incorrect verdicts show a retry message.
- Release builds receive their case configuration outside the repository. The
  solutions file is never committed.

## Requirements

- Node.js `>=26.0.0`
- pnpm `10` or newer

## Local development

Install dependencies from the repository root:

```sh
pnpm install
```

Run the unit tests and formatting check:

```sh
pnpm test
pnpm exec prettier --check .
```

To build and serve the game locally, create a local `.env.local.json` first.
The file is gitignored and contains the crime keys configured for each package;
it is intentionally not part of the public repository.

```sh
pnpm dev
```

The dev command builds the Astro site and case data, then starts the preview
server. Open:

```text
http://localhost:4321/clanker-crimes/repo?case=alpha
```

If you only need to rebuild the case data, run:

```sh
pnpm build:cases
```

A production build runs both the Astro build and the case builder:

```sh
pnpm build
```

Generated output is written to `dist/` and should not be committed. The case
builder never modifies anything under `packages/`; Vite previews are built from
temporary copies.

For a throwaway CI-style solution file, use
`node scripts/ci-solutions.mjs --force`. The `--force` flag can overwrite a real
local `.env.local.json`, so do not use it casually.

## Repository layout

```text
src/
  components/       shared Astro components
  layouts/          page shell and global metadata
  pages/            landing page and case viewer
  styles/           global styles and fonts
  data.json         public crime catalogue
packages/           playable source projects (read-only crime scenes)
scripts/build/      case-builder modules
scripts/build.mjs   case-builder entry point
tests/              builder unit tests
.github/workflows/  checks, release, and branch-retarget workflows
```

The case viewer is a client-side Astro page. Case data is served separately
from `/repo`, so do not add a page route at `/case`; the builder owns that
output directory.

## CI and releases

Pull requests and pushes to `dev` run formatting, unit tests, and a full build
with throwaway solutions. Pushes to `master` run the release workflow, build
the site, publish a release asset, and deploy `dist/` to GitHub Pages.

Keep generated data, local solution files, and build directories out of version
control. Do not add solution assignments or generated case identifiers to logs
or documentation.
