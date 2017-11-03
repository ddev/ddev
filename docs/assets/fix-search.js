// See various broken search issues, but especially:
// https://github.com/rtfd/readthedocs.org/pull/2289
// https://github.com/rtfd/readthedocs.org/issues/1088

function fixSearch() {
    var target = document.getElementById('rtd-search-form');
    var config = {attributes: true, childList: true};
  
    var observer = new MutationObserver(function(mutations) {
      observer.disconnect();
      var form = $('#rtd-search-form');
      form.empty();
      form.attr('action', 'https://' + window.location.hostname + '/en/' + determineSelectedBranch() + '/search.html');
      $('<input>').attr({
        type: "text",
        name: "q",
        placeholder: "Search docs"
      }).appendTo(form);
    });
    if (window.location.origin.indexOf('readthedocs') > -1) {
      observer.observe(target, config);
    }
  }
  
  $(document).ready(function () {
    fixSearch();
  })
