document.addEventListener("DOMContentLoaded", function () {
    load_warning_banner_for_latest();
});

function load_warning_banner_for_latest() {
    let currentUrl = window.location.href;

    if (!currentUrl.includes('/stable/')) {
        // Insert the warning banner after the nav bar
        document.querySelector('.md-tabs[data-md-component="tabs"]').insertAdjacentHTML('afterend', `
            <div id="warning-banner-for-latest" data-md-color-scheme="default" style="display: none;">
                <aside class="md-banner md-banner--warning">
                    <div class="md-banner__inner md-grid md-typeset">
                        You're viewing documentation for an unreleased version of DDEV.
                        <a id="stable-docs-link" href="https://ddev.readthedocs.io/en/stable/">
                            <strong>Click here to see the stable documentation.</strong>
                        </a>
                    </div>
                </aside>
            </div>
        `);

        // Display the warning banner after a delay to reduce flickering
        setTimeout(() => {
            document.getElementById('stable-docs-link').href = `https://ddev.readthedocs.io/${new URL(window.location.href).pathname.replace(/\/en\/[^/]+/, 'en/stable')}`;
            document.getElementById('warning-banner-for-latest').style.display = 'block';
        }, 100);
    }
}
