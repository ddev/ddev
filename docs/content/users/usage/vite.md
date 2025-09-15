# Vite Integration

[Vite](https://vitejs.dev/) is a popular frontend build tool that provides fast development server with Hot Module Replacement (HMR) and optimized production builds. DDEV supports Vite development workflows for various frameworks including Laravel, Vue.js, React, Svelte, and more.

## Quick Setup

To use Vite with DDEV, you need to:

1. **Configure DDEV to expose Vite's port** in `.ddev/config.yaml`:

    ```yaml
    web_extra_exposed_ports:
      - name: vite
        container_port: 5173
        http_port: 5172
        https_port: 5173
    ```

2. **Configure Vite** in your `vite.config.js`:

    ```javascript
    import { defineConfig } from 'vite'

    export default defineConfig({
      server: {
        // Respond to all network requests
        host: "0.0.0.0",
        port: 5173,
        strictPort: true,
        // Defines the origin of the generated asset URLs during development, this must be set to the
        // Vite dev server URL and selected port. In general, `process.env.DDEV_PRIMARY_URL` will give
        // us the primary URL of the DDEV project, e.g. "https://test-vite.ddev.site". But since DDEV
        // can be configured to use another port (via `router_https_port`), the output can also be
        // "https://test-vite.ddev.site:1234". Therefore we need to strip a port number like ":1234"
        // before adding Vites port to achieve the desired output of "https://test-vite.ddev.site:5173".
        origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
        // Configure CORS securely for the Vite dev server to allow requests from *.ddev.site domains,
        // supports additional hostnames (via regex). If you use another `project_tld`, adjust this.
        cors: {
          origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
        },
      },
    })
    ```

3. **Restart DDEV** to apply configuration changes:

    ```bash
    ddev restart
    ```

4. **Start Vite development server**:

    ```bash
    ddev npm run dev
    # or
    ddev yarn dev
    ```

Your Vite development server will be available at `https://yourproject.ddev.site:5173`.

!!! note "HTTPS Configuration"
    This guide assumes your project runs on `https://`. If you cannot access the HTTPS version of your project, see the [DDEV installation documentation](../../install/ddev-installation.md).

!!! tip "Custom TLD"
    If you use a custom `project_tld` other than `ddev.site`, adjust the CORS configuration accordingly in your `vite.config.js`.

## Framework-Specific Configuration

### Laravel

Laravel includes Vite as the default asset bundler since v9.19. Here's the recommended setup:

#### DDEV Configuration

Add to `.ddev/config.yaml`:

```yaml
web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 5172
    https_port: 5173

nodejs_version: "18"
```

#### Laravel Vite Configuration

Update your `vite.config.js`:

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
        origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
        cors: {
          origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
        },
        hmr: {
            host: "localhost",
        },
    },
});
```

#### Usage in Blade Templates

In your Blade templates, use Laravel's Vite directives:

```blade
@vite(['resources/css/app.css', 'resources/js/app.js'])
```

### Drupal

For Drupal projects using the [Vite module](https://www.drupal.org/project/vite):

#### Drupal DDEV Configuration

```yaml
web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 5172
    https_port: 5173

nodejs_version: "18"
```

#### Drupal Vite Configuration

```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/sites/default/files/vite/',
  build: {
    manifest: true,
    outDir: 'sites/default/files/vite',
    rollupOptions: {
      input: {
        main: 'assets/js/main.js',
        style: 'assets/css/style.css',
      }
    }
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
    cors: {
      origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
    },
  },
})
```

### TYPO3

For TYPO3 projects using [vite-asset-collector](https://github.com/s2b/vite-asset-collector):

#### TYPO3 DDEV Configuration

```yaml
web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 5172
    https_port: 5173

nodejs_version: "18"
```

#### TYPO3 Vite Configuration

```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/typo3temp/assets/vite/',
  build: {
    manifest: true,
    outDir: 'public/typo3temp/assets/vite/',
    rollupOptions: {
      input: {
        main: 'packages/site_package/Resources/Private/Assets/main.js',
      }
    }
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
    cors: {
      origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
    },
  },
})
```

### WordPress

For WordPress projects using Vite with themes or plugins:

#### WordPress DDEV Configuration

```yaml
web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 5172
    https_port: 5173

nodejs_version: "18"
```

#### WordPress Vite Configuration

```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/wp-content/themes/your-theme/dist/',
  build: {
    manifest: true,
    outDir: 'wp-content/themes/your-theme/dist',
    rollupOptions: {
      input: {
        main: 'wp-content/themes/your-theme/src/main.js',
      }
    }
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
    cors: {
      origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
    },
  },
})
```

### CraftCMS

For CraftCMS projects using [nystudio107's Vite plugin](https://nystudio107.com/docs/vite):

#### CraftCMS DDEV Configuration

```yaml
web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 5172
    https_port: 5173

nodejs_version: "18"
```

#### CraftCMS Vite Configuration

```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/dist/',
  build: {
    manifest: true,
    outDir: 'web/dist/',
    rollupOptions: {
      input: {
        app: 'src/js/app.js',
      }
    }
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
    cors: {
      origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
    },
    fs: {
      strict: false
    }
  },
})
```

## Advanced Configuration

### Auto-starting Vite

You can configure DDEV to automatically start Vite when the project starts using [hooks](../configuration/hooks.md):

Add to `.ddev/config.yaml`:

```yaml
hooks:
  post-start:
    - exec: "npm run dev"
      service: web
```

Or use a more robust daemon configuration:

```yaml
web_extra_daemons:
  - name: "vite"
    command: "npm run dev"
    directory: /var/www/html
```

### Production Builds

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

### DDEV Add-ons

Several community add-ons are available to simplify Vite integration:

- **[ddev-vite-sidecar](https://github.com/s2b/ddev-vite-sidecar)**: Zero-config integration of Vite into DDEV projects. The Vite development server is exposed as a `https://vite.*` subdomain, eliminating the need to expose ports to the host system.

- **[ddev-vitest](https://github.com/tyler36/ddev-vitest)**: Adds helper commands for projects using [Vitest](https://vitest.dev/), a Vite-native testing framework.

Explore the [DDEV Add-on Registry](https://addons.ddev.com) for more Vite-related add-ons.

### GitHub Codespaces

When using DDEV with GitHub Codespaces, DDEV's router is not used, so some adjustments are needed:

1. **Port exposure doesn't work via `.ddev/config.yaml`** - use a docker-compose file instead
2. **Create `.ddev/docker-compose.vite-workaround.yaml`**:

```yaml
version: '3.6'
services:
  web:
    ports:
      - "5173:5173"
```

See the [DDEV Codespaces documentation](https://docs.ddev.com/en/stable/users/install/ddev-installation/#github-codespaces) for more details.

### Using with Node.js Projects

For Node.js-only projects (like SvelteKit, Nuxt, or Vue CLI projects):

#### Node.js DDEV Configuration

```yaml
project_type: generic
webserver_type: generic

web_extra_exposed_ports:
  - name: vite
    container_port: 5173
    http_port: 80
    https_port: 443

web_extra_daemons:
  - name: "vite-dev"
    command: "npm run dev"
    directory: /var/www/html

nodejs_version: "18"
```

#### Node.js Vite Configuration

```javascript
export default defineConfig({
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    cors: {
      origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
    },
  },
})
```

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
   server: {
     cors: {
       origin: /https?:\/\/([A-Za-z0-9\-\.]+)?(\.ddev\.site)(?::\d+)?$/,
     },
   }
   ```

2. **Check origin setting**:

   ```javascript
   server: {
     origin: `${process.env.DDEV_PRIMARY_URL.replace(/:\d+$/, "")}:5173`,
   }
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
   server: {
     port: 5174,
   }
   ```

2. **Kill existing process**:

   ```bash
   ddev exec "pkill -f vite"
   ```

### HMR Not Working

**Problem**: Hot Module Replacement isn't working in the browser.

**Solutions**:

1. **Check HMR configuration**:

   ```javascript
   server: {
     hmr: {
       host: "localhost",
       port: 5173,
     },
   }
   ```

2. **For Laravel projects**, ensure you're using the `@vite` directive in your Blade templates.

3. **Check browser console** for WebSocket connection errors.

### Assets Not Loading

**Problem**: CSS/JS assets not loading properly.

**Solutions**:

1. **Verify base path** in production builds matches your web server configuration.

2. **Check manifest.json** is being generated and loaded correctly.

3. **Ensure proper asset URLs** in your templates/framework integration.

## Best Practices

1. **Use specific Node.js versions**: Specify `nodejs_version` in your DDEV configuration for consistency across team members.

2. **Include Vite in your project dependencies**: Don't rely on global Vite installations.

3. **Configure proper gitignore**: Exclude build artifacts:

   ```gitignore
   /dist/
   /build/
   node_modules/
   ```

4. **Document your setup**: Include Vite configuration instructions in your project's README.

5. **Use environment variables**: Leverage `process.env.DDEV_PRIMARY_URL` for dynamic configuration.

## Additional Resources

### WordPress Libraries

For WordPress projects, several libraries can help integrate Vite:

- [php-wordpress-vite-assets](https://github.com/idleberg/php-wordpress-vite-assets)
- [vite-for-wp](https://github.com/kucrut/vite-for-wp)
- [wp-vite-manifest](https://github.com/iamntz/wp-vite-manifest)

### Example Repositories

- [Laravel + Vite](https://github.com/mandrasch/ddev-laravel-vite)
- [WordPress + Vite Demo Theme](https://github.com/mandrasch/ddev-wp-vite-demo)
- [CraftCMS + Vite](https://github.com/mandrasch/ddev-craftcms-vite)
- [Laravel Breeze + SvelteKit Monorepo](https://github.com/mandrasch/ddev-laravel-breeze-sveltekit)

### Further Reading

- [Node.js Development with DDEV](https://www.lullabot.com/articles/nodejs-development-ddev) - Comprehensive guide to Node.js workflows in DDEV
- [How to Run Headless Drupal and NextJS on DDEV](https://www.velir.com/ideas/2024/05/13/how-to-run-headless-drupal-and-nextjs-on-ddev)
- [How we use DDEV, Vite and Tailwind with Craft CMS](https://www.viget.com/articles/how-we-use-ddev-vite-and-tailwind-with-craft-cms/)
- [Integrating Vite and DDEV into WordPress](https://www.viget.com/articles/integrating-vite-and-ddev-into-wordpress/)

## Related Documentation

- [Node.js Quickstart](../quickstart.md#nodejs)
- [Laravel Quickstart](../quickstart.md#laravel)
- [Custom Docker Services](../extend/custom-docker-services.md)
- [Networking](networking.md)
- [Hooks](../configuration/hooks.md)
- [Troubleshooting](troubleshooting.md)
