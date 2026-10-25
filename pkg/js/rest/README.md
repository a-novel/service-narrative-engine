# Service Template REST client

The TypeScript client for the template service's REST API.

## Installation

GitHub Packages serves this package. It asks for a Personal Access Token with the `read:packages` scope, even for a public package ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add the registries and your token to the `.npmrc` at your project root:

```ini
@a-novel-kit:registry=https://npm.pkg.github.com
@a-novel:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

Then install the package:

```bash
pnpm add @a-novel/service-template-rest
```

The service's [README](https://github.com/a-novel/service-template#javascript--typescript-rest) shows a first call.
