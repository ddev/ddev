const fixTabs = () => {
    // Get current hash
    let currentHash = window.location.href.split('#')[1];
    if (!currentHash) return

    // Get element associated with hash
    let hashElement = document.getElementById(currentHash);
    if (!hashElement) return

    // Detect if element is located in a tab
    let parentElement = hashElement.closest('.tabbed-block');
    if (!parentElement) return

    // Get all tab labels
    let allTabs = hashElement.closest('.tabbed-set').querySelector('.tabbed-labels').children;
    if (!allTabs) return

    // get index of tab to click on
    let index = [...parentElement.parentNode.children].indexOf(parentElement)

    // Simulate mouse click click on our tab label
    allTabs[index].click();

    // If our tab is nested within another tab, open parent tab..
    if (allTabs[0].getAttribute('for').startsWith('__tabbed_2')) {
        // Get outer tabs
        let outerTabs = document.querySelector('.tabbed-labels').children

        // Get outer tab block
        let outerParent = allTabs[0].closest('.tabbed-block');

        // Get outer tab index
        let outerIndex = [...outerParent.parentNode.children].indexOf(outerParent)

        // Active outer tab
        outerTabs[outerIndex].click();
    }

    if (allTabs[0].getAttribute('for').startsWith('__tabbed_3')) {
        console.log('While 1 nested tab is debatable 2 should be avoided as it is definitely not user friendly.')
    }

    // scroll to hash
    hashElement.scrollIntoView();
};

// Fires after initial page load
window.addEventListener('load', fixTabs, false);

// Fires whenever the hash changes
window.addEventListener('hashchange', fixTabs, false);