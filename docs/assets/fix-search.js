(function (){
  var MutationObserver = (function () {
    var prefixes = ['WebKit', 'Moz', 'O', 'Ms', '']
    for (var i=0; i < prefixes.length; i++) {
      if (prefixes[i] + 'MutationObserver' in window) {
        return window[prefixes[i] + 'MutationObserver'];
      }
    }
    return false;
  }());

  /*
  * RTD messes up MkDocs' search feature by tinkering with the search box defined in the theme, see
  * https://github.com/rtfd/readthedocs.org/issues/1088. This function sets up a DOM4 MutationObserver
  * to react to changes to the search form (triggered by RTD on doc ready). It then reverts everything
  * the RTD JS code modified.
  *
  * @see https://github.com/rtfd/readthedocs.org/issues/1088#issuecomment-224715045
  */
  $(document).ready(function () {
    if (!MutationObserver) {
      return;
    }
    var target = document.getElementById('rtd-search-form');
    var config = {attributes: true, childList: true};

    var observer = new MutationObserver(function(mutations) {
      // if it isn't disconnected it'll loop infinitely because the observed element is modified
      observer.disconnect();
      var form = $('#rtd-search-form');
      var path = window.location.pathname;
      var branch = path.split('/')[2];
      form.empty();
      form.attr('action', window.location.origin + '/en/' + branch + '/search.html');
      $('<input>').attr({
        type: "text",
        name: "q",
        placeholder: "Search docs"
      }).appendTo(form);
    });

    observer.observe(target, config);
  });
}());
