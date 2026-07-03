const openTabsFromHash = () => {
    // Get current hash
    let currentHash = window.location.href.split('#')[1];
    if (!currentHash) return

    // Get element associated with hash
    let hashElement = document.getElementById(currentHash);
    if (!hashElement) return

    // Walk up through every enclosing tab (handles tabs nested to any depth)
    // and activate each one, from the innermost to the outermost.
    let block = hashElement.closest('.tabbed-block');
    while (block) {
        let tabbedSet = block.closest('.tabbed-set');
        if (!tabbedSet) break;

        // The labels of this tab set, in the same order as its content blocks.
        let labels = tabbedSet.querySelector('.tabbed-labels');
        if (labels) {
            let index = [...block.parentNode.children].indexOf(block);
            let label = labels.children[index];
            if (label) label.click();
        }

        // Continue from the block that encloses this tab set, if any.
        block = tabbedSet.parentNode ? tabbedSet.parentNode.closest('.tabbed-block') : null;
    }

    // scroll to hash
    hashElement.scrollIntoView();
};

// Fires after initial page load
window.addEventListener('load', openTabsFromHash, false);

// Fires whenever the hash changes
window.addEventListener('hashchange', openTabsFromHash, false);
