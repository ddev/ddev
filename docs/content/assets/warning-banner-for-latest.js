document.addEventListener("DOMContentLoaded", function () {
    load_warning_banner_for_latest();
});

function load_warning_banner_for_latest() {
    let currentUrl = window.location.href;

    if (currentUrl.includes('/latest/')) {
        // Insert the warning banner as the first element in the body
        document.body.insertAdjacentHTML('afterbegin', `
            <div id="warning-banner-for-latest" data-md-color-scheme="default" style="display: none;">
                <aside class="md-banner md-banner--warning">
                    <div class="md-banner__inner md-grid md-typeset">
                        You’re viewing the latest unreleased version.
                        <a href="${currentUrl.replace(/\/latest\//, '/stable/')}">
                            <strong>Click here to go to stable.</strong>
                        </a>
                    </div>
                </aside>
            </div>
        `);

        // Display the warning banner after a delay to reduce flickering
        setTimeout(() => {
            let warningBanner = document.getElementById('warning-banner-for-latest');
            if (warningBanner) {
                warningBanner.style.display = 'block';
            }
        }, 100);
    }
}
