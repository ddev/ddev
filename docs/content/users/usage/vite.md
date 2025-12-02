# Vite Integration

[Vite](https://vitejs.dev/) is a popular front-end build tool that provides fast development server with Hot Module Replacement (HMR) and optimized production builds. DDEV supports Vite development workflows for various frameworks including Laravel, Vue.js, React, Svelte, and more.

## Quick Setup

To use Vite with DDEV, you need to:

1. Configure DDEV to expose Vite's port in `.ddev/config.yaml`:

    ```yaml
    web_extra_exposed_ports:
      - name: vite
        container_port: 5173
        http_port: 5172
        https_port: 5173
    ```

2. Configure Vite in your `vite.config.js`:

    ```javascript
    import { defineConfig } from 'vite'

    export default defineConfig({
      // Your settings
      // ...

      // Adjust Vites dev server to work with DDEV
      // https://vitejs.dev/config/server-options.html
      server: {
        // Respond to all network requests
        host: "0.0.0.0",
        port: 5173,
        strictPort: true,
        // Defines the origin of the generated asset URLs during development,
        // this must be set to the Vite dev server URL and selected port.
        origin: `${process.env.DDEV_PRIMARY_URL_WITHOUT_PORT}:5173`,
        // Configure CORS securely for the Vite dev server to allow requests
        // from *.ddev.site domains, supports additional hostnames (via regex).
        // If you use another `project_tld`, adjust this value accordingly.
        cors: {
          origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
        },
      },
    })
    ```

3. Restart DDEV to apply configuration changes:

    ```bash
    ddev restart
    ```

4. Start Vite development server:

    ```bash
    ddev npm run dev
    # or
    ddev yarn dev
    ```

Your Vite development server will be available at `https://yourproject.ddev.site:5173`.

!!!note "HTTPS Configuration"
    This guide assumes your project runs on `https://`. If you are unable to access the HTTPS version of your project, refer to the [Configuring Browsers](../install/configuring-browsers.md).

!!!tip "Custom TLD"
    If you use a custom [`project_tld`](../configuration/config.md#project_tld) other than `ddev.site`, adjust the CORS configuration accordingly in your `vite.config.js`, or use this snippet:

    ```javascript
    export default defineConfig({
      server: {
        cors: {
          origin: new RegExp(
            `https?:\/\/(${process.env.DDEV_HOSTNAME.split(",")
              .map((h) => h.replace("*", "[^.]+"))
              .join("|")})(?::\\d+)?$`
          )
        },
      },
    });
    ```

## Example Projects

Example implementations demonstrating Vite integration with DDEV:

- **[Working with Vite in DDEV](https://ddev.com/blog/working-with-vite-in-ddev/)** - Basic PHP project with step-by-step setup guide
- **[vite-php-setup](https://github.com/andrefelipe/vite-php-setup)** - General PHP + Vite integration (adapt `VITE_HOST` for DDEV)

For additional integration patterns and framework-specific examples:

- **[Vite Backend Integration Guide](https://vitejs.dev/guide/backend-integration.html)** - Official Vite documentation for backend frameworks
- **[Vite Awesome List](https://github.com/vitejs/awesome-vite#integrations-with-backends)** - Community-maintained list of integrations

## Craft CMS

The [Vite plugin by `nystudio107`](https://nystudio107.com/docs/vite/) provides official DDEV support with detailed [configuration instructions](https://nystudio107.com/docs/vite/#using-ddev) for `vite.config.js` and `config/vite.php`.

!!!note "Port Configuration"
    When using `web_extra_exposed_ports` in `.ddev/config.yaml`, the `.ddev/docker-compose.*.yaml` file for port exposure is not required.

Example implementations:

- **[Craft CMS with Vite integration](https://github.com/mandrasch/ddev-craftcms-vite)**
- **[Craft CMS Starter](https://github.com/vigetlabs/craft-site-starter)**
- **[How we use DDEV, Vite and Tailwind with Craft CMS](https://www.viget.com/articles/how-we-use-ddev-vite-and-tailwind-with-craft-cms/)**

## Drupal

Several tools and modules are available for integrating Vite with Drupal:

- **[Vite module](https://www.drupal.org/project/vite)** - Uses Vite's manifest.json to map Drupal library files to compiled versions in `/dist` or to the Vite development server
- **[UnoCSS Starter theme](https://www.drupal.org/project/unocss_starter)** - Drupal theme with Vite integration and DDEV setup instructions
- **[Foxy](https://www.drupal.org/project/foxy)** - Alternative asset bundling solution for Drupal

Community resources:

- **[UnoCSS Starter Theme Documentation](https://www.drupalarchitect.info/projects/unocss-starter-theme)**
- **[Proof of concept for Vite bundling in Drupal](https://github.com/darvanen/drupal-js)**

The Drupal community is actively developing Vite integration solutions for bundling assets across multiple modules and themes.

## Laravel

Laravel adopted [Vite as the default asset bundler](https://laravel-news.com/vite-is-the-default-frontend-asset-bundler-for-laravel-applications) in v9.19, replacing Laravel Mix.

Configure DDEV to expose Vite's port in `.ddev/config.yaml`:

```yaml
web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 5172
    https_port: 5173
```

Configure Vite in your `vite.config.js`:

```javascript
import { defineConfig } from 'vite';
import laravel from 'laravel-vite-plugin';

export default defineConfig({
  plugins: [
    laravel({
      input: [
        'resources/css/app.css',
        'resources/js/app.js',
      ],
      refresh: true,
    }),
  ],
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    origin: `${process.env.DDEV_PRIMARY_URL_WITHOUT_PORT}:5173`,
    cors: {
      origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
    },
  },
});
```

Start the Vite development server:

```bash
ddev npm run dev
```

Example implementation:

- **[ddev-laravel-vite](https://github.com/mandrasch/ddev-laravel-vite)** - Laravel with Vite integration

!!!note "Laravel-Specific Integration"
    Laravel's Vite integration uses a `public/hot` file to manage development server state through its npm integration.

## Node.js

DDEV supports Node.js-only projects by proxying requests to the correct ports within the web container. This configuration enables running Node.js applications like Keystone CMS or SvelteKit entirely within DDEV.

This approach supports various architectures:

- **Monorepo setup**: Run a PHP backend with a Node.js frontend on separate subdomains within a single DDEV project
- **Headless CMS**: Ideal for decoupled architectures combining traditional backends with modern JavaScript frameworks
- **Multi-project setup**: Use separate DDEV projects for frontend and backend with [inter-project communication](../usage/managing-projects.md#inter-project-communication)

Additional resources:

- **[Node.js Development with DDEV](https://www.lullabot.com/articles/nodejs-development-ddev)** - Comprehensive guide to Node.js project configuration
- **[How to Run Headless Drupal and Next.js on DDEV](https://www.velir.com/ideas/2024/05/13/how-to-run-headless-drupal-and-nextjs-on-ddev)** - Headless CMS implementation guide
- **[ddev-laravel-breeze-sveltekit](https://github.com/mandrasch/ddev-laravel-breeze-sveltekit)** - Monorepo example with Laravel and SvelteKit

## TYPO3

Several tools are available for integrating Vite with TYPO3:

- **[typo3-vite-demo](https://github.com/fgeierst/typo3-vite-demo)** - Vite demo project
- **[vite-asset-collector](https://github.com/s2b/vite-asset-collector)** - TYPO3 extension for Vite integration
- **[vite-plugin-typo3](https://github.com/s2b/vite-plugin-typo3)** - Vite plugin for TYPO3
- **[ddev-vite-sidecar](https://github.com/s2b/ddev-vite-sidecar)** - DDEV add-on for zero-config Vite integration

The vite-asset-collector extension provides detailed [DDEV installation instructions](https://docs.typo3.org/p/praetorius/vite-asset-collector/main/en-us/Installation/Index.html#installation-1). For questions or support, join the Vite channel on [TYPO3 Slack](https://typo3.community/meet/slack).

## WordPress

Several libraries are available for integrating Vite with WordPress:

- **[php-wordpress-vite-assets](https://github.com/idleberg/php-wordpress-vite-assets)** - PHP library for loading Vite assets in WordPress
- **[vite-for-wp](https://github.com/kucrut/vite-for-wp)** - WordPress integration for Vite
- **[wp-vite-manifest](https://github.com/iamntz/wp-vite-manifest)** - Vite manifest loader for WordPress

Example implementations:

- **[ddev-wp-vite-demo](https://github.com/mandrasch/ddev-wp-vite-demo)** - Demo theme using php-wordpress-vite-assets
- **[wp-vite-manifest usage guide](https://github.com/iamntz/wp-vite-manifest/wiki/How-to-use-inside-your-WP-plugin-theme#usage)** - Integration guide for themes and plugins
- **[Integrating Vite and DDEV into WordPress](https://www.viget.com/articles/integrating-vite-and-ddev-into-wordpress/)**

## GitHub Codespaces

DDEV supports Vite in GitHub Codespaces with alternative port configuration. Example implementations:

- **[Laravel with Vite in Codespaces](https://github.com/mandrasch/ddev-laravel-vite)**
- **[Craft CMS with Vite in Codespaces](https://github.com/mandrasch/ddev-craftcms-vite)**

!!!note "Alternative Port Configuration Required"
    The DDEV router is unavailable in Codespaces. Instead of using `web_extra_exposed_ports` in `.ddev/config.yaml`, create a `.ddev/docker-compose.vite-workaround.yaml` file:

    ```yaml
    services:
      web:
        ports:
          - 5173:5173
    ```

For additional Codespaces configuration details, see the [DDEV Codespaces documentation](../install/ddev-installation.md#github-codespaces).

## Auto-starting Vite

You can configure DDEV to automatically start Vite when the project starts using [hooks](../configuration/hooks.md):

Add to `.ddev/config.yaml`:

```yaml
hooks:
  post-start:
    - exec: "npm run dev"
```

Or use a more robust daemon configuration (logs available via `ddev logs -s web`):

```yaml
web_extra_daemons:
  - name: "vite"
    command: bash -c 'npm install && npm run dev -- --host'
    directory: /var/www/html
```

For a real-world daemon implementation example, see the [ddev.com repository](https://github.com/ddev/ddev.com) configuration.

## Production Builds

For production builds, ensure your `vite.config.js` includes proper manifest generation:

```javascript
export default defineConfig({
  build: {
    manifest: true,
    rollupOptions: {
      input: {
        main: 'path/to/your/main.js',
      }
    }
  },
  // ... other configuration
})
```

Build for production:

```bash
ddev npm run build
```

## DDEV Add-ons

Several community add-ons simplify Vite integration:

- **[ddev-vite-sidecar](https://github.com/s2b/ddev-vite-sidecar)** - Zero-config Vite integration exposing the development server as a `https://vite.*` subdomain, eliminating the need to expose ports to the host system
- **[ddev-vitest](https://github.com/tyler36/ddev-vitest)** - Helper commands for projects using [Vitest](https://vitest.dev/), a Vite-native testing framework
- **[ddev-viteserve](https://github.com/torenware/ddev-viteserve)** - First DDEV Vite add-on (no longer maintained, but pioneered the integration)

Additional Vite-related add-ons are available in the [DDEV Add-on Registry](https://addons.ddev.com).

## Troubleshooting

### Bad Gateway Errors

**Problem**: Getting "502 Bad Gateway" when accessing Vite URL.

**Solutions**:

1. **Check port configuration**: Ensure `web_extra_exposed_ports` is correctly configured in `.ddev/config.yaml`.

2. **Verify Vite is running**: Check if Vite development server is actually running:

    ```bash
    ddev logs -s web
    ```

3. **Restart DDEV**: After changing configuration:

    ```bash
    ddev restart
    ```

### CORS Issues

**Problem**: Browser console shows CORS errors.

**Solutions**:

1. **Update CORS configuration** in `vite.config.js`:

    ```javascript
    export default defineConfig({
      server: {
        cors: {
          origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
        },
      },
    });
    ```

2. **Check origin setting**:

    ```javascript
    export default defineConfig({
      server: {
        origin: `${process.env.DDEV_PRIMARY_URL_WITHOUT_PORT}:5173`,
      },
    });
    ```

### Port Already in Use

**Problem**: "Port 5173 is already in use" error.

**Solutions**:

1. **Use different port**: Change the port in both DDEV and Vite configurations:

    ```yaml
    # .ddev/config.yaml
    web_extra_exposed_ports:
      - name: vite
        container_port: 5174
        http_port: 5172
        https_port: 5174
    ```

    ```javascript
    // vite.config.js
    export default defineConfig({
      server: {
        port: 5174,
      },
    });
    ```

2. **Kill existing process**:

    ```bash
    ddev exec "pkill -f vite"
    ```

### Assets Not Loading

**Problem**: CSS/JS assets not loading properly.

**Solutions**:

1. **Verify base path** in production builds matches your web server configuration.

2. **Check manifest.json** is being generated and loaded correctly.

3. **Ensure proper asset URLs** in your templates/framework integration.

## Best Practices

1. **Use specific Node.js versions**: Specify `nodejs_version` in your DDEV configuration for consistency across team members.

2. **Include Vite in your project dependencies**: Don't rely on global Vite installations.

3. **Configure proper `.gitignore`**: Exclude build artifacts:

    ```gitignore
    /dist/
    /build/
    node_modules/
    ```

4. **Document your setup**: Include Vite configuration instructions in your project's readme.

5. **Use environment variables**: Leverage `process.env.DDEV_PRIMARY_URL_WITHOUT_PORT` for dynamic configuration.
